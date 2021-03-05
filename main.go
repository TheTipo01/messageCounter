package main

import (
	"database/sql"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

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

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			lit.Error("Config file not found! See example_config.yml")
			return
		}
	} else {
		// Config file found
		token = viper.GetString("token")

		// Set lit.LogLevel to the given value
		switch strings.ToLower(viper.GetString("loglevel")) {
		case "logerror", "error":
			lit.LogLevel = lit.LogError
			break
		case "logwarning", "warning":
			lit.LogLevel = lit.LogWarning
			break
		case "loginformational", "informational":
			lit.LogLevel = lit.LogInformational
			break
		case "logdebug", "debug":
			lit.LogLevel = lit.LogDebug
			break
		}

		// Open database connection
		db, err = sql.Open(viper.GetString("drivername"), viper.GetString("datasourcename"))
		if err != nil {
			lit.Error("Error opening db connection, %s", err)
			return
		}

		// Initialize tables
		execQuery(tblMessages)
		execQuery(tblUsers)
		execQuery(tblServer)
		execQuery(tblChannels)
		execQuery(tblPings)
		execQuery(tblConfig)

		// And add the everyone user to the table, as we use that for logging @everyone and @here
		stm, _ := db.Prepare("INSERT INTO users (id, nickname) VALUES(?, ?)")
		_, err = stm.Exec("everyone", "everyone")
		if err != nil {
			str := err.Error()
			if !strings.HasPrefix(str, "Error 1062: Duplicate entry") {
				lit.Error("Error inserting user everyone in the database, %s", str)
			}
		}
		_ = stm.Close()

	}

}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		lit.Error("error creating Discord session, %s", err)
		return
	}

	dg.AddHandler(messageCreate)
	dg.AddHandler(messageDelete)
	dg.AddHandler(messageUpdate)
	dg.AddHandler(guildCreate)
	dg.AddHandler(ready)

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

	if server[m.GuildID].numberOfMessages++; server[m.GuildID].numberOfMessages == 69419 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "nice")
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
					lit.Info("Finished getting messages for #%s in \"%s\"", c.Name, g.Name)
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
		lit.Info("Added offset of %d on guild \"%s\". New total of message %d", offset, g.Name, server[g.ID].numberOfMessages)
	}
}

func ready(s *discordgo.Session, _ *discordgo.Ready) {
	// Set the playing status.
	err := s.UpdateGameStatus(0, "ghostping.ga")
	if err != nil {
		lit.Error("Can't set status, %s", err)
	}
}
