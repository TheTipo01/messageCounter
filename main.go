package main

import (
	"database/sql"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kkyr/fig"
	"github.com/mb-14/gomarkov"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Config holds data parsed from the config.yml
type Config struct {
	Token    string `fig:"token" validate:"required"`
	Driver   string `fig:"drivername" validate:"required"`
	DSN      string `fig:"datasourcename" validate:"required"`
	LogLevel string `fig:"loglevel" validate:"required"`
	Site     string `fig:"site" validate:"required"`
}

var (
	// Discord bot token
	token string
	// Database connection
	db *sql.DB
	// Server structure for all the things we need (currently only the number of messages)
	server = make(map[string]*Server)
	// Site URL
	site string
)

func init() {
	lit.LogLevel = lit.LogError

	var cfg Config
	err := fig.Load(&cfg, fig.File("config.yml"), fig.Dirs(".", "./data"))
	if err != nil {
		lit.Error(err.Error())
		return
	}

	token = cfg.Token
	site = cfg.Site

	// Set lit.LogLevel to the given value
	switch strings.ToLower(cfg.LogLevel) {
	case "logwarning", "warning":
		lit.LogLevel = lit.LogWarning
	case "loginformational", "informational":
		lit.LogLevel = lit.LogInformational
	case "logdebug", "debug":
		lit.LogLevel = lit.LogDebug
	}

	// Open database connection
	db, err = sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		lit.Error("Error opening db connection, %s", err)
		return
	}

	db.SetConnMaxLifetime(time.Minute * 3)

	// Initialize tables
	execQuery(tblMessages, tblUsers, tblServers, tblChannels, tblPings, tblConfig, tblPollState, tblPollsGroups)

	// And add the everyone user to the table, as we use that for logging @everyone and @here
	_, err = db.Exec("INSERT IGNORE INTO users (id, nickname) VALUES(?, ?)", "everyone", "everyone")
	if err != nil {
		lit.Error("Error inserting user everyone in the database, %s", err.Error())
	}
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		lit.Error("error creating Discord session, %s", err)
		return
	}

	// Add events handler
	dg.AddHandler(messageCreate)
	dg.AddHandler(messageDelete)
	dg.AddHandler(messageUpdate)
	dg.AddHandler(guildCreate)
	dg.AddHandler(ready)
	dg.AddHandler(reactionAdd)
	dg.AddHandler(reactionRemove)

	// Add commands handler
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// Ignores commands from DM
		if i.User == nil {
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				c := make(chan struct{})
				go func() {
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
					})
					c <- struct{}{}
				}()

				h(s, i, c)
			}
		}
	})

	// Initialize intents that we use
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages | discordgo.IntentsGuilds | discordgo.IntentsGuildMessageReactions)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		lit.Error("error opening connection, %s", err)
		return
	}

	// Register commands
	_, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands)
	if err != nil {
		lit.Error("Can't register commands, %s", err)
	}

	loadScheduler(dg)
	loadModel()
	getHiddenChannels()

	// Wait here until CTRL-C or other term signal is received.
	lit.Info("messageCounter is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()

	saveAllModels()

	// And the database connection
	_ = db.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	addMessage(m.Message)

	server[m.GuildID].numberOfMessages++

	switch server[m.GuildID].numberOfMessages {
	case 69419:
		_, _ = s.ChannelMessageSend(m.ChannelID, "nice")
	case 99999:
		_, _ = s.ChannelMessageSend(m.ChannelID, "So guys, we did it. We finally reached 100k messages")
	}

	if server[m.GuildID].hiddenChannel == m.ChannelID {
		for _, a := range m.Attachments {
			if (a.ContentType == "image/png" || a.ContentType == "image/jpeg") && !strings.HasPrefix(a.Filename, "SPOILER") {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Hey "+m.Author.Mention()+", are you sure you want to post this here without a spoiler tag?")
			}
		}
	}
}

func messageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	server[m.GuildID].numberOfMessages--
	deleteMessage(s, m.Message)
}

func messageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
	updateMessage(s, m.Message)
}

// Gets old messages that the bot missed when it was offline/just added to a new guild
func guildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	var (
		err      error
		messages []*discordgo.Message
		afterID  string
		offset   int
	)

	if server[g.ID] == nil {
		_, err = db.Exec("INSERT INTO servers (id, name, model) VALUES(?, ?, '')", g.ID, g.Name)
		if err != nil {
			lit.Error("Error inserting into the database: %s", err.Error())
		}

		server[g.ID] = &Server{numberOfMessages: 0, model: gomarkov.NewChain(1)}
	}

	for _, c := range g.Channels {
		if c.Type != discordgo.ChannelTypeGuildVoice && c.Type != discordgo.ChannelTypeGuildCategory {
			for {
				_ = db.QueryRow("SELECT messageID FROM messages WHERE guildID=? AND channelID=? ORDER BY JSON_VALUE(message, '$.timestamp') DESC LIMIT 1", c.GuildID, c.ID).Scan(&afterID)
				messages, err = s.ChannelMessages(c.ID, 100, "", afterID, "")
				if err != nil {
					lit.Error("error while getting messages, %s", err)
					break
				}

				for _, m := range messages {
					m.GuildID = c.GuildID
					addMessage(m)
				}

				if len(messages) < 100 {
					lit.Debug("Finished getting messages for #%s in \"%s\"", c.Name, g.Name)
					break
				}
			}

			afterID = ""
		}
	}

	// Initialize count of message
	_ = db.QueryRow("SELECT COUNT(*) FROM messages WHERE guildID=? AND deleted=0", g.ID).Scan(&server[g.ID].numberOfMessages)

	// Add offset of message, so that the notification for the message number 69420 is sent correctly.
	// Sometime we have duplicate message.
	_ = db.QueryRow("SELECT offset FROM config WHERE guildID=?", g.ID).Scan(&offset)
	if offset != 0 {
		server[g.ID].numberOfMessages += offset
		lit.Debug("Added offset of %d on guild \"%s\". New total of message %d", offset, g.Name, server[g.ID].numberOfMessages)
	}

	saveModel(g.ID)
}

func ready(s *discordgo.Session, _ *discordgo.Ready) {
	// Set the playing status.
	err := s.UpdateGameStatus(0, site)
	if err != nil {
		lit.Error("Can't set status, %s", err)
	}
}

func reactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}

	reactionUpdate(s, r.MessageReaction, false)
}

func reactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.UserID == s.State.User.ID {
		return
	}

	reactionUpdate(s, r.MessageReaction, true)
}

func reactionUpdate(s *discordgo.Session, r *discordgo.MessageReaction, removed bool) {
	// Checks if the message is a poll
	if _, ok := server[r.GuildID].polls[r.MessageID]; ok && (r.Emoji.Name == "👍" || r.Emoji.Name == "👎") {
		var tmp, question, userAnswered, userAnsweredPositive string
		// Checks if the user is in the group for that poll
		_ = db.QueryRow("SELECT userIDs, question, userAnswered, userAnsweredPositive FROM pollState, pollsGroup WHERE messageID = ? AND serverID = ? AND pollsGroup.name = pollState.groupName", r.MessageID, r.GuildID).Scan(&tmp, &question, &userAnswered, &userAnsweredPositive)

		if tmp != "" {
			// Group found
			userIDs := strings.Split(tmp, ",")

			// Checks if the user is in the group
			for _, id := range userIDs {
				if id == r.UserID {
					// User found, we modify the message
					var userAnsweredPositiveUpdated, userAnsweredUpdated []string

					// Returns if the user already answered
					if strings.Contains(userAnswered, r.UserID) && !removed {
						break
					}

					// Removed or adds the user accordingly
					if removed {
						userAnsweredUpdated = removeString(strings.Split(userAnswered, ","), r.UserID)
					} else {
						userAnsweredUpdated = append(strings.Split(userAnswered, ","), r.UserID)
					}

					cleanSlice(&userAnsweredUpdated)
					answerNumber := len(userAnsweredUpdated)

					if r.Emoji.Name == "👍" {
						// Add or remove the user from the positive answer
						if removed {
							userAnsweredPositiveUpdated = removeString(strings.Split(userAnsweredPositive, ","), r.UserID)
						} else {
							userAnsweredPositiveUpdated = append(strings.Split(userAnsweredPositive, ","), r.UserID)
						}
					} else {
						userAnsweredPositiveUpdated = strings.Split(userAnsweredPositive, ",")
					}

					cleanSlice(&userAnsweredPositiveUpdated)

					embed := NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", question).
						AddField("Answered", "Number of people who answered: "+strconv.Itoa(answerNumber)).
						AddField("Remaining", "People who still need to answer: "+formatUsers(userIDs, userAnsweredUpdated)).
						AddField("Percentage", "Percentage of people who answered positively: "+strconv.Itoa(int((float64(len(userAnsweredPositiveUpdated))/float64(answerNumber))*100))+"%").
						SetColor(0x00ff00).MessageEmbed

					// We update the message embed
					_, _ = s.ChannelMessageEditEmbed(r.ChannelID, r.MessageID, embed)

					// And the database
					_, _ = db.Exec("UPDATE pollState SET userAnswered = ?, userAnsweredPositive = ? WHERE messageID = ?", strings.Join(userAnsweredUpdated, ","), strings.Join(userAnsweredPositiveUpdated, ","), r.MessageID)
					break
				}
			}
		}
	}
}
