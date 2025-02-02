package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	"github.com/go-co-op/gocron"
	"github.com/goccy/go-json"
	"strings"
	"time"
)

const (
	tblMessages    = "CREATE TABLE IF NOT EXISTS `messages` ( `guildID` varchar(20) NOT NULL DEFAULT '0', `channelID` varchar(20) NOT NULL DEFAULT '0', `messageID` varchar(20) CHARACTER SET utf8mb4 NOT NULL, `authorID` varchar(20) DEFAULT NULL, `message` longtext CHARACTER SET utf8mb4 NOT NULL CHECK (json_valid(`message`)), `deleted` tinyint(1) unsigned NOT NULL DEFAULT 0, PRIMARY KEY (`messageID`) USING BTREE ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
	tblUsers       = "CREATE TABLE IF NOT EXISTS `users` ( `id` varchar(20) NOT NULL, `nickname` varchar(32) NOT NULL, PRIMARY KEY (`id`) ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
	tblServers     = "CREATE TABLE IF NOT EXISTS `servers` ( `id` varchar(20) NOT NULL, `name` varchar(100) NOT NULL, `model` longtext NOT NULL, PRIMARY KEY (`id`) ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
	tblChannels    = "CREATE TABLE IF NOT EXISTS `channels` ( `id` varchar(20) NOT NULL, `name` text NOT NULL DEFAULT '', `serverId` varchar(20) NOT NULL, PRIMARY KEY (`id`), KEY `FK_channels_server` (`serverId`), CONSTRAINT `FK_channels_server` FOREIGN KEY (`serverId`) REFERENCES `servers` (`id`) ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
	tblPings       = "CREATE TABLE IF NOT EXISTS `pings` ( `id` int(11) NOT NULL AUTO_INCREMENT, `menzionatoreId` varchar(20) NOT NULL, `menzionatoId` varchar(20) NOT NULL, `channelId` varchar(20) NOT NULL, `serverId` varchar(20) NOT NULL, `timestamp` datetime NOT NULL, `messageId` varchar(20) NOT NULL, PRIMARY KEY (`id`), KEY `FK_pings_channels` (`channelId`), KEY `FK_pings_server` (`serverId`), KEY `FK_pings_users` (`menzionatoreId`), KEY `FK_pings_users_2` (`menzionatoId`), KEY `messageId` (`messageId`), CONSTRAINT `FK_pings_channels` FOREIGN KEY (`channelId`) REFERENCES `channels` (`id`), CONSTRAINT `FK_pings_messages` FOREIGN KEY (`messageId`) REFERENCES `messages` (`messageID`), CONSTRAINT `FK_pings_server` FOREIGN KEY (`serverId`) REFERENCES `servers` (`id`), CONSTRAINT `FK_pings_users` FOREIGN KEY (`menzionatoreId`) REFERENCES `users` (`id`), CONSTRAINT `FK_pings_users_2` FOREIGN KEY (`menzionatoId`) REFERENCES `users` (`id`) ) ENGINE=InnoDB AUTO_INCREMENT=88 DEFAULT CHARSET=utf8mb4;"
	tblConfig      = "CREATE TABLE IF NOT EXISTS `config` ( `id` int(11) NOT NULL AUTO_INCREMENT, `guildID` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0', `channelID` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0', `channelToID` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0', `offset` int(11) DEFAULT 0, `hiddenChannel` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL, PRIMARY KEY (`id`), KEY `FK_config_server` (`guildID`), KEY `FK_config_channels` (`channelID`), KEY `FK_config_channels_2` (`channelToID`), KEY `FK_config_channels_3` (`hiddenChannel`), CONSTRAINT `FK_config_channels` FOREIGN KEY (`channelID`) REFERENCES `channels` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION, CONSTRAINT `FK_config_channels_2` FOREIGN KEY (`channelToID`) REFERENCES `channels` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION, CONSTRAINT `FK_config_channels_3` FOREIGN KEY (`hiddenChannel`) REFERENCES `channels` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION, CONSTRAINT `FK_config_server` FOREIGN KEY (`guildID`) REFERENCES `servers` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION ) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_general_ci;"
	tblPollState   = "CREATE TABLE IF NOT EXISTS `pollState` ( `messageID` varchar(20) NOT NULL, `userAnswered` text NOT NULL DEFAULT '', `question` text NOT NULL, `groupName` varchar(255) NOT NULL, `guildID` varchar(20) NOT NULL, `userAnsweredPositive` text NOT NULL DEFAULT '', PRIMARY KEY (`messageID`) ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
	tblPollsGroups = "CREATE TABLE IF NOT EXISTS `pollsGroup` ( `serverID` varchar(20) NOT NULL, `name` varchar(255) NOT NULL, `userIDs` text NOT NULL DEFAULT '', `createdBy` varchar(20) NOT NULL, PRIMARY KEY (`serverID`,`name`), CONSTRAINT `FK_groups_servers` FOREIGN KEY (`serverID`) REFERENCES `servers` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
)

// Executes a simple query given a DB
func execQuery(query ...string) {
	for _, q := range query {
		_, err := db.Exec(q)
		if err != nil {
			lit.Error("Error executing query, %s", err)
			return
		}
	}
}

// addMessage adds a message to the db
func addMessage(m *discordgo.Message) {
	var err error
	inJSON, _ := json.Marshal(m)

	if m.Author != nil {
		insertAuthor(m)
		_, err = db.Exec("INSERT INTO messages (guildID, channelID, messageID, authorID, message) VALUES (?, ?, ?, ?, ?)", m.GuildID, m.ChannelID, m.ID, m.Author.ID, inJSON)

		if !m.Author.Bot {
			server[m.GuildID].model.Add(strings.Split(m.Content, " "))
		}
	} else {
		_, err = db.Exec("INSERT INTO messages (guildID, channelID, messageID, message) VALUES (?, ?, ?, ?)", m.GuildID, m.ChannelID, m.ID, inJSON)
	}

	if err != nil {
		lit.Error("Error while inserting message into db, %s", err)
	}

}

func deleteMessage(s *discordgo.Session, m *discordgo.Message) {
	// Set delete flag up
	_, err := db.Exec("UPDATE messages SET deleted=1 WHERE messageID=?", m.ID)
	if err != nil {
		lit.Error("Error updating row from the database, %s", err)
	}

	// Add mentions to the pings table
	var (
		message    []byte
		oldMessage discordgo.Message
	)

	_ = db.QueryRow("SELECT message FROM messages WHERE messageID=?", m.ID).Scan(&message)
	_ = json.Unmarshal(message, &oldMessage)

	if oldMessage.MentionEveryone {
		insertData(s, &oldMessage, nil)

		_, err = db.Exec("INSERT INTO pings (menzionatoreId, menzionatoId, channelId, serverId, timestamp, messageId) VALUES(?, ?, ?, ?, NOW(), ?)",
			oldMessage.Author.ID, "everyone", oldMessage.ChannelID, oldMessage.GuildID, oldMessage.ID)
		if err != nil {
			lit.Error("Error inserting row in the database, %s", err)
		}
	} else {
		if len(oldMessage.Mentions) > 0 {
			for _, mention := range oldMessage.Mentions {
				insertData(s, &oldMessage, mention)

				_, err = db.Exec("INSERT INTO pings (menzionatoreId, menzionatoId, channelId, serverId, timestamp, messageId) VALUES(?, ?, ?, ?, NOW(), ?)",
					oldMessage.Author.ID, mention.ID, oldMessage.ChannelID, oldMessage.GuildID, oldMessage.ID)
				if err != nil {
					lit.Error("Error inserting row in the database, %s", err)
				}
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

	_ = db.QueryRow("SELECT message FROM messages WHERE messageID=?", m.ID).Scan(&message)
	_ = json.Unmarshal(message, &oldMessage)

	// Update existing message
	jsonMessage, _ := json.Marshal(m)

	_, err := db.Exec("UPDATE messages SET message=? WHERE messageID=?", jsonMessage, m.ID)
	if err != nil {
		lit.Error("Error updating row from the database, %s", err)
	}

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

			_, err = db.Exec("INSERT INTO pings (menzionatoreId, menzionatoId, channelId, serverId, timestamp, messageId) VALUES(?, ?, ?, ?, NOW(), ?)",
				m.Author.ID, oldM.ID, m.ChannelID, m.GuildID, m.ID)
			if err != nil {
				lit.Error("Error inserting row in the database, %s", err)
			}
		} else {
			found = false
		}
	}

	// If the state of mentionEveryone changed, that's a ghostping of everyone
	if !m.MentionEveryone && oldMessage.MentionEveryone {
		insertData(s, &oldMessage, nil)

		_, err = db.Exec("INSERT INTO pings (menzionatoreId, menzionatoId, channelId, serverId, timestamp, messageId) VALUES(?, ?, ?, ?, NOW(), ?)", m.Author.ID, "everyone", m.ChannelID, m.GuildID, m.ID)
		if err != nil {
			lit.Error("Error inserting row in the database, %s", err)
		}
	}
}

// Populates channels, guilds and users tables
func insertData(s *discordgo.Session, message *discordgo.Message, mention *discordgo.User) {
	var err error

	// Guild
	g, err := s.Guild(message.GuildID)
	if err == nil {
		_, err = db.Exec("INSERT IGNORE INTO servers (id, name) VALUES(?, ?)", g.ID, g.Name)
		if err != nil {
			lit.Error("Error inserting channel in the database, %s", err)
		}
	} else {
		lit.Error("cannot create guild, %s", err)
	}

	// Author insert
	insertAuthor(message)

	// Mentioned
	if mention != nil {
		_, err = db.Exec("INSERT IGNORE INTO users (id, nickname) VALUES(?, ?)", mention.ID, mention.Username)
		if err != nil {
			lit.Error("Error inserting user in the database, %s", err)
		}
	}

	// Channel
	channel, err := s.Channel(message.ChannelID)
	if err == nil {
		_, err = db.Exec("INSERT IGNORE INTO channels (id, name, serverId) VALUES(?, ?, ?)", channel.ID, channel.Name, channel.GuildID)
		if err != nil {
			lit.Error("Error inserting channel in the database, %s", err)
		}
	} else {
		lit.Error("cannot create channel, %s", err)
	}

}

func insertAuthor(message *discordgo.Message) {
	if message.Author != nil {
		_, err := db.Exec("INSERT IGNORE INTO users (id, nickname) VALUES(?, ?)", message.Author.ID, message.Author.Username)
		if err != nil {
			lit.Error("Error inserting user in the database, %s", err.Error())
		}
	}
}

// Every Monday at midnight sends a random message for configured guilds
func loadScheduler(s *discordgo.Session) {
	var guildID, channelID, channelToID string

	// Create cron scheduler
	cron := gocron.NewScheduler(time.Local)

	config, err := db.Query("SELECT guildID, channelID, channelToID FROM config")
	if err != nil {
		lit.Error("Can't query database, %s", err)
		return
	}

	for config.Next() {
		err = config.Scan(&guildID, &channelID, &channelToID)
		if err != nil {
			lit.Error("Can't scan config, %s", err)
			continue
		}

		// Send random message from a channel every monday at midnight
		_, _ = cron.Every(1).Monday().At("00:00:00").Do(sendQuoteToServer, s, guildID, channelID, channelToID)

		lit.Debug("Added cronjob for server %s", guildID)
	}

	_, _ = cron.Every(5).Minute().Do(saveAllModels)

	// And start the scheduler
	cron.StartAsync()
}

func sendQuoteToServer(s *discordgo.Session, guildID, channelID, channelToID string) {
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

	// If there's an attachments, add it
	if len(message.Attachments) > 0 {
		message.Content = message.Attachments[0].URL + "\n" + message.Content
	}

	rows := strings.Split(message.Content, "\n")
	quoted_msg := ""
	for _, row := range rows {
		quoted_msg += "> " + row
	}

	_, err = s.ChannelMessageSend(channelToID, "Quote of the week:\n"+quoted_msg+"\nSubmitted by "+message.Author.Username)
	if err != nil {
		lit.Error("Can't send message, %s", err)
		return
	}
}

// Returns the nickname of a person given its ID
func getNickname(id string) string {
	var nickname string

	err := db.QueryRow("SELECT nickname FROM users WHERE id=?", id).Scan(&nickname)
	if err != nil {
		lit.Error("Can't get nickname, %s", err)
		return ""
	}

	return nickname
}

func getHiddenChannels() {
	var guildID, channelID string

	rows, _ := db.Query("SELECT guildID, hiddenChannel FROM config WHERE hiddenChannel IS NOT NULL")

	for rows.Next() {
		err := rows.Scan(&guildID, &channelID)
		if err != nil {
			lit.Error("Can't get hidden channels, %s", err)
			continue
		}

		server[guildID].hiddenChannel = channelID
	}
}
