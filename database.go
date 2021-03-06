package main

import (
	"database/sql"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	"github.com/go-co-op/gocron"
	"github.com/goccy/go-json"
	"strings"
	"time"
)

const (
	tblMessages = "CREATE TABLE IF NOT EXISTS `messages`( `guildID` varchar(18) CHARACTER SET utf8mb4 NOT NULL, `channelID` varchar(18) CHARACTER SET utf8mb4 NOT NULL, `messageID` varchar(18) CHARACTER SET utf8mb4 NOT NULL, `message` text CHARACTER SET utf8mb4 NOT NULL, `deleted` tinyint(1) NOT NULL DEFAULT 0, PRIMARY KEY (`messageID`) USING BTREE) DEFAULT CHARSET=utf8;"
	tblUsers    = "CREATE TABLE IF NOT EXISTS `users`( `id` varchar(18) NOT NULL, `nickname` varchar(32) NOT NULL, PRIMARY KEY (`id`)) DEFAULT CHARSET=utf8mb4;"
	tblServer   = "CREATE TABLE IF NOT EXISTS `server`( `id` varchar(18) NOT NULL, `name` varchar(100) NOT NULL, PRIMARY KEY (`id`)) DEFAULT CHARSET=utf8mb4;"
	tblChannels = "CREATE TABLE IF NOT EXISTS `channels`( `id` varchar(18) NOT NULL, `name` text NOT NULL DEFAULT '', `serverId` varchar(18) NOT NULL, PRIMARY KEY (`id`), KEY `FK_channels_server` (`serverId`), CONSTRAINT `FK_channels_server` FOREIGN KEY (`serverId`) REFERENCES `server` (`id`)) DEFAULT CHARSET=utf8mb4;"
	tblPings    = "CREATE TABLE IF NOT EXISTS `pings`( `id` int(11) NOT NULL AUTO_INCREMENT, `menzionatoreId` varchar(18) NOT NULL, `menzionatoId` varchar(18) NOT NULL, `channelId` varchar(18) NOT NULL, `serverId` varchar(18) NOT NULL, `timestamp` datetime NOT NULL, `messageId` varchar(18) NOT NULL, PRIMARY KEY (`id`), KEY `FK_pings_channels` (`channelId`), KEY `FK_pings_server` (`serverId`), KEY `FK_pings_users` (`menzionatoreId`), KEY `FK_pings_users_2` (`menzionatoId`), KEY `messageId` (`messageId`), CONSTRAINT `FK_pings_channels` FOREIGN KEY (`channelId`) REFERENCES `channels` (`id`), CONSTRAINT `FK_pings_messages` FOREIGN KEY (`messageId`) REFERENCES `messages` (`messageID`), CONSTRAINT `FK_pings_server` FOREIGN KEY (`serverId`) REFERENCES `server` (`id`), CONSTRAINT `FK_pings_users` FOREIGN KEY (`menzionatoreId`) REFERENCES `users` (`id`), CONSTRAINT `FK_pings_users_2` FOREIGN KEY (`menzionatoId`) REFERENCES `users` (`id`)) DEFAULT CHARSET=utf8mb4;"
	tblConfig   = "CREATE TABLE IF NOT EXISTS `config`( `id` int(11) NOT NULL AUTO_INCREMENT, `guildID` varchar(18) CHARACTER SET utf8mb4 NOT NULL DEFAULT '0', `channelID` varchar(18) CHARACTER SET utf8mb4 NOT NULL DEFAULT '0', `channelToID` varchar(18) CHARACTER SET utf8mb4 NOT NULL DEFAULT '0', `offset` int(11) DEFAULT 0, PRIMARY KEY (`id`), KEY `FK_config_server` (`guildID`), KEY `FK_config_channels` (`channelID`), KEY `FK_config_channels_2` (`channelToID`), CONSTRAINT `FK_config_channels` FOREIGN KEY (`channelID`) REFERENCES `channels` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION, CONSTRAINT `FK_config_channels_2` FOREIGN KEY (`channelToID`) REFERENCES `channels` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION, CONSTRAINT `FK_config_server` FOREIGN KEY (`guildID`) REFERENCES `server` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4;"
)

// Executes a simple query given a DB
func execQuery(query string) {
	statement, err := db.Prepare(query)
	if err != nil {
		lit.Error("Error preparing query, %s", err)
		return
	}

	_, err = statement.Exec()
	if err != nil {
		lit.Error("Error creating table, %s", err)
	}
}

// addMessage adds a message to the db
func addMessage(m *discordgo.Message) {
	stm, _ := db.Prepare("INSERT INTO messages (guildID, channelID, messageID, message) VALUES (?, ?, ?, ?)")

	inJSON, _ := json.Marshal(m)

	_, err := stm.Exec(m.GuildID, m.ChannelID, m.ID, string(inJSON))
	if err != nil {
		lit.Error("Error while inserting message into db, %s", err)
	}

	_ = stm.Close()
}

func deleteMessage(s *discordgo.Session, m *discordgo.Message) {
	// Set delete flag up
	stm, _ := db.Prepare("UPDATE messages SET deleted=1 WHERE guildID=? AND channelID=? AND messageID=?")
	_, err := stm.Exec(m.GuildID, m.ChannelID, m.ID)
	if err != nil {
		lit.Error("Error updating row from the database, %s", err)
	}

	_ = stm.Close()

	// Add mentions to the pings table
	var (
		message    []byte
		oldMessage discordgo.Message
	)

	_ = db.QueryRow("SELECT message FROM messages WHERE guildID=? AND channelID=? AND messageID=?", m.GuildID, m.ChannelID, m.ID).Scan(&message)
	_ = json.Unmarshal(message, &oldMessage)

	if oldMessage.MentionEveryone {
		insertData(s, &oldMessage, nil)

		stm, _ = db.Prepare("INSERT INTO pings (menzionatoreId, menzionatoId, channelId, serverId, timestamp, messageId) VALUES(?, ?, ?, ?, NOW(), ?)")
		_, err = stm.Exec(oldMessage.Author.ID, "everyone", oldMessage.ChannelID, oldMessage.GuildID, oldMessage.ID)
		if err != nil {
			lit.Error("Error inserting row in the database, %s", err)
		}

		_ = stm.Close()
	} else {
		if len(oldMessage.Mentions) > 0 {
			for _, mention := range oldMessage.Mentions {
				insertData(s, &oldMessage, mention)

				stm, _ = db.Prepare("INSERT INTO pings (menzionatoreId, menzionatoId, channelId, serverId, timestamp, messageId) VALUES(?, ?, ?, ?, NOW(), ?)")
				_, err = stm.Exec(oldMessage.Author.ID, mention.ID, oldMessage.ChannelID, oldMessage.GuildID, oldMessage.ID)
				if err != nil {
					lit.Error("Error inserting row in the database, %s", err)
				}

				_ = stm.Close()
			}
		}
	}
}

func updateMessage(s *discordgo.Session, m *discordgo.Message) {
	// Get old message, to compare mentions
	var (
		message    []byte
		oldMessage discordgo.Message
	)

	_ = db.QueryRow("SELECT message FROM messages WHERE guildID=? AND channelID=? AND messageID=?", m.GuildID, m.ChannelID, m.ID).Scan(&message)
	_ = json.Unmarshal(message, &oldMessage)

	// Update existing message
	jsonMessage, _ := json.Marshal(m)

	stm, _ := db.Prepare("UPDATE messages SET message=? WHERE guildID=? AND channelID=? AND messageID=?")
	_, err := stm.Exec(string(jsonMessage), m.GuildID, m.ChannelID, m.ID)
	if err != nil {
		lit.Error("Error updating row from the database, %s", err)
	}

	_ = stm.Close()

	// Compare mentions
	var (
		found = false
	)

	// If the ping didn't change to @everyone, we check deeply
	for _, oldM := range oldMessage.Mentions {
		for _, newM := range m.Mentions {
			if newM.ID == oldM.ID {
				found = true
				break
			}
		}

		if !found {
			// User was ghostpinged, we add that to the database
			insertData(s, m, oldM)

			stm, _ = db.Prepare("INSERT INTO pings (menzionatoreId, menzionatoId, channelId, serverId, timestamp, messageId) VALUES(?, ?, ?, ?, NOW(), ?)")
			_, err = stm.Exec(m.Author.ID, oldM.ID, m.ChannelID, m.GuildID, m.ID)
			if err != nil {
				lit.Error("Error inserting row in the database, %s", err)
			}

			_ = stm.Close()
		} else {
			found = false
		}
	}

	// If the state of mentionEveryone changed, that's a ghostping of everyone
	if !m.MentionEveryone && oldMessage.MentionEveryone {
		insertData(s, &oldMessage, nil)

		stm, _ = db.Prepare("INSERT INTO pings (menzionatoreId, menzionatoId, channelId, serverId, timestamp, messageId) VALUES(?, ?, ?, ?, NOW(), ?)")
		_, err = stm.Exec(m.Author.ID, "everyone", m.ChannelID, m.GuildID, m.ID)
		if err != nil {
			lit.Error("Error inserting row in the database, %s", err)
		}

		_ = stm.Close()
	}
}

// Populates channels, guilds and users tables
func insertData(s *discordgo.Session, message *discordgo.Message, mention *discordgo.User) {
	var (
		err error
		stm *sql.Stmt
		str string
	)

	// Guild
	g, err := s.Guild(message.GuildID)
	if err == nil {
		stm, _ = db.Prepare("INSERT INTO server (id, name) VALUES(?, ?)")
		_, err = stm.Exec(g.ID, g.Name)
		if err != nil {
			str = err.Error()
			if !strings.HasPrefix(str, "Error 1062: Duplicate entry") {
				lit.Error("Error inserting channel in the database, %s", str)
			}
		}

		_ = stm.Close()
	} else {
		lit.Error("cannot create guild, %s", err)
	}

	// Author insert
	stm, _ = db.Prepare("INSERT INTO users (id, nickname) VALUES(?, ?)")
	_, err = stm.Exec(message.Author.ID, message.Author.Username)
	if err != nil {
		str = err.Error()
		if !strings.HasPrefix(str, "Error 1062: Duplicate entry") {
			lit.Error("Error inserting user in the database, %s", str)
		}
	}

	_ = stm.Close()

	// Mentioned
	if mention != nil {
		stm, _ = db.Prepare("INSERT INTO users (id, nickname) VALUES(?, ?)")
		_, err = stm.Exec(mention.ID, mention.Username)
		if err != nil {
			str = err.Error()
			if !strings.HasPrefix(str, "Error 1062: Duplicate entry") {
				lit.Error("Error inserting user in the database, %s", str)
			}
		}

		_ = stm.Close()
	}

	// Channel
	channel, err := s.Channel(message.ChannelID)
	if err == nil {
		stm, _ = db.Prepare("INSERT INTO channels (id, name, serverId) VALUES(?, ?, ?)")
		_, err = stm.Exec(channel.ID, channel.Name, channel.GuildID)
		if err != nil {
			str = err.Error()
			if !strings.HasPrefix(str, "Error 1062: Duplicate entry") {
				lit.Error("Error inserting channel in the database, %s", str)
			}
		}

		_ = stm.Close()
	} else {
		lit.Error("cannot create channel, %s", err)
	}

}

// Every Monday at midnight sends a random message for configured guilds
func loadScheduler(s *discordgo.Session) {
	// Create cron scheduler
	cron := gocron.NewScheduler(time.Local)

	config, err := db.Query("SELECT guildID, channelID, channelToID FROM config")
	if err != nil {
		lit.Error("Can't query database, %s", err)
		return
	}

	for config.Next() {
		var guildID, channelID, channelToID string
		err = config.Scan(&guildID, &channelID, &channelToID)
		if err != nil {
			lit.Error("Can't scan config, %s", err)
			return
		}

		// Send random message from a channel every monday at midnight
		_, _ = cron.Every(1).Monday().At("00:00:00").Do(func() {
			var (
				messageJSON []byte
				message     discordgo.Message
				err         error
			)

			err = db.QueryRow("SELECT message FROM messages WHERE guildID=? AND channelID=? AND deleted = 0 ORDER BY RAND() LIMIT 1", guildID, channelID).Scan(&messageJSON)
			if err != nil {
				lit.Error("Can't get random message, %s", err)
				return
			}

			err = json.Unmarshal(messageJSON, &message)
			if err != nil {
				lit.Error("Can't unmarshall message, %s", err)
				return
			}

			_, err = s.ChannelMessageSend(channelToID, "Quote of the week:```\n"+message.Content+"```Submitted by "+message.Author.Username)
			if err != nil {
				lit.Error("Can't send message, %s", err)
				return
			}
		})

		lit.Debug("Added cronjob for server %s", guildID)
	}

	// And start the scheduler
	cron.StartAsync()
}
