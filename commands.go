package main

import (
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	jsoniter "github.com/json-iterator/go"
	"regexp"
)

var (
	// Commands
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "characters",
			Description: "Prints the number of characters sent for a channel, or the entire server if omitted",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Optional channel to get stats for",
					Required:    false,
				},
			},
		},
		{
			Name:        "words",
			Description: "Prints the number of words sent for a channel, or the entire server if omitted",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Optional channel to get stats for",
					Required:    false,
				},
			},
		},
		{
			Name:        "messages",
			Description: "Prints the number of messages sent for a channel, or the entire server if omitted",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Optional channel to get stats for",
					Required:    false,
				},
			},
		},
		{
			Name:        "charsPerMex",
			Description: "Prints the number of characters per message sent for a channel, or the entire server if omitted",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Optional channel to get stats for",
					Required:    false,
				},
			},
		},
	}

	// Handler
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		// Prints the number of characters sent for a given channel, or if not specified for the entire server
		"characters": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var (
				mex                 *sql.Rows
				err                 error
				json                = jsoniter.ConfigFastest
				messageJSON, toSend string
				m                   discordgo.Message
				cont                int
				characters          = make(map[string]int)
				people              = make(map[string]string)
			)

			// If there's a specified channel, use it in the query
			if len(i.Data.Options) > 0 {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=? AND channelID=?", i.GuildID, i.Data.Options[0].ChannelValue(s).ID)
			} else {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=?", i.GuildID)
			}
			if err != nil {
				lit.Error("Can't query database, %s", err)
				return
			}

			for mex.Next() {
				err = mex.Scan(&messageJSON)
				if err != nil {
					lit.Error("Can't scan m, %s", err)
					return
				}

				err = json.Unmarshal([]byte(messageJSON), &m)
				if err != nil {
					lit.Error("Can't unmarshal JSON, %s", err)
					continue
				}

				if m.Author != nil {
					characters[m.Author.ID] += len(m.Content)
					people[m.Author.ID] = m.Author.Username
				}
			}

			// Characters
			for i, kv := range sorting(characters) {
				cont += kv.Value
				toSend += fmt.Sprintf("%d) %s: %d\n", i+1, people[kv.Key], kv.Value)
			}
			toSend = fmt.Sprintf("Number of characters sent: %d\n\n", cont) + toSend

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Characters", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)
		},

		// Prints the number of words sent for a given channel, or if not specified for the entire server
		"words": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var (
				mex                 *sql.Rows
				err                 error
				json                = jsoniter.ConfigFastest
				messageJSON, toSend string
				m                   discordgo.Message
				cont                int
				words               = make(map[string]int)
				people              = make(map[string]string)
				// Match non-space character sequences.
				re = regexp.MustCompile(`[\S]+`)
			)

			// If there's a specified channel, use it in the query
			if len(i.Data.Options) > 0 {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=? AND channelID=?", i.GuildID, i.Data.Options[0].ChannelValue(s).ID)
			} else {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=?", i.GuildID)
			}
			if err != nil {
				lit.Error("Can't query database, %s", err)
				return
			}

			for mex.Next() {
				err = mex.Scan(&messageJSON)
				if err != nil {
					lit.Error("Can't scan m, %s", err)
					return
				}

				err = json.Unmarshal([]byte(messageJSON), &m)
				if err != nil {
					lit.Error("Can't unmarshal JSON, %s", err)
					continue
				}

				if m.Author != nil {
					people[m.Author.ID] = m.Author.Username
					words[m.Author.ID] += len(re.FindAllString(m.Content, -1))
				}
			}

			// Words
			for i, kv := range sorting(words) {
				cont += kv.Value
				toSend += fmt.Sprintf("%d) %s: %d\n", i+1, people[kv.Key], kv.Value)
			}
			toSend = fmt.Sprintf("Number of words: %d\n\n", cont) + toSend

			sendEmbedInteractionFollowup(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Words", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)
		},

		// Prints the number of messages sent for a given channel, or if not specified for the entire server
		"messages": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var (
				mex                 *sql.Rows
				err                 error
				json                = jsoniter.ConfigFastest
				messageJSON, toSend string
				m                   discordgo.Message
				cont                int
				people              = make(map[string]string)
				messages            = make(map[string]int)
			)

			// If there's a specified channel, use it in the query
			if len(i.Data.Options) > 0 {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=? AND channelID=?", i.GuildID, i.Data.Options[0].ChannelValue(s).ID)
			} else {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=?", i.GuildID)
			}
			if err != nil {
				lit.Error("Can't query database, %s", err)
				return
			}

			for mex.Next() {
				err = mex.Scan(&messageJSON)
				if err != nil {
					lit.Error("Can't scan m, %s", err)
					return
				}

				err = json.Unmarshal([]byte(messageJSON), &m)
				if err != nil {
					lit.Error("Can't unmarshal JSON, %s", err)
					continue
				}

				if m.Author != nil {
					people[m.Author.ID] = m.Author.Username
					messages[m.Author.ID]++
				}
			}

			// Messages
			for i, kv := range sorting(messages) {
				cont += kv.Value
				toSend += fmt.Sprintf("%d) %s: %d\n", i+1, people[kv.Key], kv.Value)
			}
			toSend = fmt.Sprintf("Number of messages: %d\n\n", cont) + toSend

			sendEmbedInteractionFollowup(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Messages", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)
		},

		// Prints stats for a given channel, or if not specified for the entire server.
		"charsPerMex": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var (
				mex                 *sql.Rows
				err                 error
				json                = jsoniter.ConfigFastest
				messageJSON, toSend string
				m                   discordgo.Message
				cont                int
				characters          = make(map[string]int)
				people              = make(map[string]string)
				messages            = make(map[string]int)
				charPerMex          = make(map[string]int)
			)

			// If there's a specified channel, use it in the query
			if len(i.Data.Options) > 0 {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=? AND channelID=?", i.GuildID, i.Data.Options[0].ChannelValue(s).ID)
			} else {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=?", i.GuildID)
			}
			if err != nil {
				lit.Error("Can't query database, %s", err)
				return
			}

			for mex.Next() {
				err = mex.Scan(&messageJSON)
				if err != nil {
					lit.Error("Can't scan m, %s", err)
					return
				}

				err = json.Unmarshal([]byte(messageJSON), &m)
				if err != nil {
					lit.Error("Can't unmarshal JSON, %s", err)
					continue
				}

				if m.Author != nil {
					characters[m.Author.ID] += len(m.Content)
					people[m.Author.ID] = m.Author.Username
					messages[m.Author.ID]++
				}
			}

			// Characters
			for k, v := range characters {
				charPerMex[k] = v / messages[k]
			}

			for i, kv := range sorting(charPerMex) {
				cont += kv.Value
				toSend += fmt.Sprintf("%d) %s: %d\n", i+1, people[kv.Key], kv.Value)
			}
			toSend = fmt.Sprintf("Number of characters per message sent: %d\n\n", cont) + toSend

			sendEmbedInteractionFollowup(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Characters per message", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)

		},
	}
)
