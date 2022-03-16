package main

import (
	"database/sql"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kkyr/fig"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// Config holds data parsed from the config.yml
type Config struct {
	Token    string `fig:"token" validate:"required"`
	Driver   string `fig:"drivername" validate:"required"`
	DSN      string `fig:"datasourcename" validate:"required"`
	LogLevel string `fig:"loglevel" validate:"required"`
}

var (
	// Discord bot token
	token string
	// Database connection
	db *sql.DB
	// Server structure for all the things we need (currently only the number of messages)
	server = make(map[string]*Server)
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

	// Initialize tables
	execQuery(tblMessages, tblUsers, tblServers, tblChannels, tblPings, tblConfig)

	// And add the everyone user to the table, as we use that for logging @everyone and @here
	_, err = db.Exec("INSERT INTO users (id, nickname) VALUES(?, ?)", "everyone", "everyone")
	if err != nil {
		str := err.Error()
		if !strings.HasPrefix(str, "Error 1062: Duplicate entry") {
			lit.Error("Error inserting user everyone in the database, %s", str)
		}
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

	// Add commands handler
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// Ignores commands from DM
		if i.User == nil {
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		}
	})

	// Initialize intents that we use
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages | discordgo.IntentsGuilds)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		lit.Error("error opening connection, %s", err)
		return
	}

	loadScheduler(dg)

	// Wait here until CTRL-C or other term signal is received.
	lit.Info("messageCounter is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()
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
		beforeID string
		offset   int
	)

	if server[g.ID] == nil {
		server[g.ID] = &Server{numberOfMessages: 0}
	}

	for _, c := range g.Channels {
		if c.Type != discordgo.ChannelTypeGuildVoice && c.Type != discordgo.ChannelTypeGuildCategory {

			for {
				_ = db.QueryRow("SELECT messageID FROM messages WHERE guildID=? AND channelID=? ORDER BY messageID LIMIT 1", c.GuildID, c.ID).Scan(&beforeID)
				messages, err = s.ChannelMessages(c.ID, 100, beforeID, "", "")
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

			beforeID = ""
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

	server[g.ID].model = loadModel(g.ID)
	if server[g.ID].model == nil {
		lit.Info("Model for guild %s doesn't exist. Building it right now", g.ID)
		server[g.ID].model = buildModel(g.ID)
	}

	saveModel(g.ID)
}

func ready(s *discordgo.Session, _ *discordgo.Ready) {
	// Set the playing status.
	err := s.UpdateGameStatus(0, "ghostping.ga")
	if err != nil {
		lit.Error("Can't set status, %s", err)
	}

	// Checks for unused commands and deletes them
	if cmds, err := s.ApplicationCommands(s.State.User.ID, ""); err == nil {
		found := false

		for _, l := range commands {
			found = false

			for _, o := range cmds {
				// We compare every online command with the ones locally stored, to find if a command with the same name exists
				if l.Name == o.Name {
					// If the options of the command are not equal, we re-register it
					if !isCommandEqual(l, o) {
						lit.Info("Registering command `%s`", l.Name)

						_, err = s.ApplicationCommandCreate(s.State.User.ID, "", l)
						if err != nil {
							lit.Error("Cannot create '%s' command: %s", l.Name, err)
						}
					}

					found = true
					break
				}
			}

			// If we didn't found a match for the locally stored command, it means the command is new. We register it
			if !found {
				lit.Info("Registering new command `%s`", l.Name)

				_, err = s.ApplicationCommandCreate(s.State.User.ID, "", l)
				if err != nil {
					lit.Error("Cannot create '%s' command: %s", l.Name, err)
				}
			}
		}
	}
}
