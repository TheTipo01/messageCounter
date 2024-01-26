package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	"github.com/goccy/go-json"
	"github.com/mb-14/gomarkov"
	"github.com/psykhi/wordclouds"
	"image/color"
	"image/png"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
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
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "includebot",
					Description: "Whether to include or not message from bots",
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
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "includebot",
					Description: "Whether to include or not message from bots",
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
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "includebot",
					Description: "Whether to include or not message from bots",
					Required:    false,
				},
			},
		},
		{
			Name:        "charspermex",
			Description: "Prints the number of characters per message sent for a channel, or the entire server if omitted",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Optional channel to get stats for",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "includebot",
					Description: "Whether to include or not message from bots",
					Required:    false,
				},
			},
		},
		{
			Name:        "wordcloud",
			Description: "Generates a word cloud for a channel, or the entire server if omitted",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Optional channel to get stats for",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "includebot",
					Description: "Whether to include or not message from bots",
					Required:    false,
				},
			},
		},
		{
			Name:        "undelete",
			Description: "Recovers the last n delete messages from the current channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "number",
					Description: "How many messages to recover",
					Required:    false,
				},
			},
		},
		{
			Name:        "markov",
			Description: "Generates a message from the current markov chain",
		},
		{
			Name:        "longest",
			Description: "Links the longest messages",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Optional channel for the stat",
					Required:    false,
				},
			},
		},
		{
			Name:        "poll",
			Description: "Creates a poll",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "question",
					Description: "The question to ask",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "group",
					Description: "A group of people to poll",
					Required:    true,
				},
			},
		},
		{
			Name:        "creategroup",
			Description: "Creates a group of people to poll",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "The name of the group",
					Required:    true,
				},
			},
		},
		{
			Name:        "deletegroup",
			Description: "Deletes a group of people to poll",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "The name of the group",
					Required:    true,
				},
			},
		},
		{
			Name:        "addmember",
			Description: "Adds a member to a group",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "group",
					Description: "The selected group",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "member",
					Description: "The member to add",
					Required:    true,
				},
			},
		},
		{
			Name:        "removemember",
			Description: "Removes a member to a group",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "group",
					Description: "The selected group",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "member",
					Description: "The member to remove",
					Required:    true,
				},
			},
		},
	}

	// Handler
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}){
		// Prints the number of characters sent for a given channel, or if not specified for the entire server
		"characters": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var (
				mex         *sql.Rows
				err         error
				messageJSON []byte
				toSend      string
				m           LightMessage
				cont        int
				characters  = make(map[string]int)
				people      = make(map[string]string)
				channel     string
				bot         bool
			)

			for _, o := range i.ApplicationCommandData().Options {
				switch o.Type {
				case discordgo.ApplicationCommandOptionChannel:
					channel = o.ChannelValue(nil).ID
				case discordgo.ApplicationCommandOptionBoolean:
					bot = o.BoolValue()
				}
			}

			// If there's a specified channel, use it in the query
			if channel != "" {
				mex, err = db.Query("SELECT message FROM messages WHERE channelID=?", channel)
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

				err = json.Unmarshal(messageJSON, &m)
				if err != nil {
					lit.Error("Can't unmarshal JSON, %s", err)
					continue
				}

				if m.Author.ID != "" {
					if bot {
						characters[m.Author.ID] += len(m.Content)
						people[m.Author.ID] = m.Author.Username
					} else {
						if !m.Author.Bot {
							characters[m.Author.ID] += len(m.Content)
							people[m.Author.ID] = m.Author.Username
						}
					}
				}
			}

			// Characters
			for i, kv := range sorting(characters) {
				cont += kv.Value
				toSend += fmt.Sprintf("%d) %s: %d\n", i+1, people[kv.Key], kv.Value)
			}
			toSend = fmt.Sprintf("Number of characters sent: %d\n\n", cont) + toSend

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Characters", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction, c)
		},

		// Prints the number of words sent for a given channel, or if not specified for the entire server
		"words": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var (
				mex         *sql.Rows
				err         error
				messageJSON []byte
				toSend      string
				m           LightMessage
				cont        int
				words       = make(map[string]int)
				people      = make(map[string]string)
				// Match non-space character sequences.
				re      = regexp.MustCompile(`\S+`)
				channel string
				bot     bool
			)

			for _, o := range i.ApplicationCommandData().Options {
				switch o.Type {
				case discordgo.ApplicationCommandOptionChannel:
					channel = o.ChannelValue(nil).ID
				case discordgo.ApplicationCommandOptionBoolean:
					bot = o.BoolValue()
				}
			}

			// If there's a specified channel, use it in the query
			if channel != "" {
				mex, err = db.Query("SELECT message FROM messages WHERE channelID=?", channel)
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

				err = json.Unmarshal(messageJSON, &m)
				if err != nil {
					lit.Error("Can't unmarshal JSON, %s", err)
					continue
				}

				if m.Author.ID != "" {
					if bot {
						people[m.Author.ID] = m.Author.Username
						words[m.Author.ID] += len(re.FindAllString(m.Content, -1))
					} else {
						if !m.Author.Bot {
							people[m.Author.ID] = m.Author.Username
							words[m.Author.ID] += len(re.FindAllString(m.Content, -1))
						}
					}
				}
			}

			// Words
			for i, kv := range sorting(words) {
				cont += kv.Value
				toSend += fmt.Sprintf("%d) %s: %d\n", i+1, people[kv.Key], kv.Value)
			}
			toSend = fmt.Sprintf("Number of words: %d\n\n", cont) + toSend

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Words", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction, c)
		},

		// Prints the number of messages sent for a given channel, or if not specified for the entire server
		"messages": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var (
				mex         *sql.Rows
				err         error
				messageJSON []byte
				toSend      string
				m           LightMessage
				cont        int
				people      = make(map[string]string)
				messages    = make(map[string]int)
				channel     string
				bot         bool
			)

			for _, o := range i.ApplicationCommandData().Options {
				switch o.Type {
				case discordgo.ApplicationCommandOptionChannel:
					channel = o.ChannelValue(nil).ID
				case discordgo.ApplicationCommandOptionBoolean:
					bot = o.BoolValue()
				}
			}

			// If there's a specified channel, use it in the query
			if channel != "" {
				mex, err = db.Query("SELECT message FROM messages WHERE channelID=?", channel)
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

				err = json.Unmarshal(messageJSON, &m)
				if err != nil {
					lit.Error("Can't unmarshal JSON, %s", err)
					continue
				}

				if m.Author.ID != "" {
					if bot {
						people[m.Author.ID] = m.Author.Username
						messages[m.Author.ID]++
					} else {
						if !m.Author.Bot {
							people[m.Author.ID] = m.Author.Username
							messages[m.Author.ID]++
						}
					}
				}
			}

			// Messages
			for i, kv := range sorting(messages) {
				cont += kv.Value
				toSend += fmt.Sprintf("%d) %s: %d\n", i+1, people[kv.Key], kv.Value)
			}
			toSend = fmt.Sprintf("Number of messages: %d\n\n", cont) + toSend

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Messages", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction, c)
		},

		// Prints stats for a given channel, or if not specified for the entire server.
		"charspermex": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var (
				mex         *sql.Rows
				err         error
				messageJSON []byte
				toSend      string
				m           LightMessage
				cont        int
				characters  = make(map[string]int)
				people      = make(map[string]string)
				messages    = make(map[string]int)
				charPerMex  = make(map[string]int)
				channel     string
				bot         bool
			)

			for _, o := range i.ApplicationCommandData().Options {
				switch o.Type {
				case discordgo.ApplicationCommandOptionChannel:
					channel = o.ChannelValue(nil).ID
				case discordgo.ApplicationCommandOptionBoolean:
					bot = o.BoolValue()
				}
			}

			// If there's a specified channel, use it in the query
			if channel != "" {
				mex, err = db.Query("SELECT message FROM messages WHERE channelID=?", channel)
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

				err = json.Unmarshal(messageJSON, &m)
				if err != nil {
					lit.Error("Can't unmarshal JSON, %s", err)
					continue
				}

				if m.Author.ID != "" {
					if bot {
						characters[m.Author.ID] += len(m.Content)
						people[m.Author.ID] = m.Author.Username
						messages[m.Author.ID]++
					} else {
						if !m.Author.Bot {
							characters[m.Author.ID] += len(m.Content)
							people[m.Author.ID] = m.Author.Username
							messages[m.Author.ID]++
						}
					}
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

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Characters per message", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction, c)

		},

		// Generates a word cloud for a channel, or the entire server if omitted
		"wordcloud": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var (
				mex         *sql.Rows
				err         error
				messageJSON []byte
				m           LightMessage
				words       = make(map[string]int)
				channel     string
				bot         bool
			)

			for _, o := range i.ApplicationCommandData().Options {
				switch o.Type {
				case discordgo.ApplicationCommandOptionChannel:
					channel = o.ChannelValue(nil).ID
				case discordgo.ApplicationCommandOptionBoolean:
					bot = o.BoolValue()
				}
			}

			// If there's a specified channel, use it in the query
			if channel != "" {
				mex, err = db.Query("SELECT message FROM messages WHERE channelID=?", channel)
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
					continue
				}

				err = json.Unmarshal(messageJSON, &m)
				if err != nil {
					lit.Error("Can't unmarshal JSON, %s", err)
					continue
				}

				if bot {
					mSplitted := strings.Fields(strings.ToLower(m.Content))
					for _, word := range mSplitted {
						if utf8.RuneCountInString(word) > 3 {
							words[word]++
						}
					}
				} else {
					if m.Author.ID != "" && !m.Author.Bot {
						mSplitted := strings.Fields(strings.ToLower(m.Content))
						for _, word := range mSplitted {
							if utf8.RuneCountInString(word) > 3 {
								words[word]++
							}
						}
					}
				}
			}

			w := wordclouds.NewWordcloud(
				words,
				wordclouds.FontFile("./fonts/Roboto-Regular.ttf"),
				wordclouds.Height(2048),
				wordclouds.Width(2048),
				wordclouds.Colors([]color.Color{color.RGBA{R: 247, G: 144, B: 30, A: 255}, color.RGBA{R: 194, G: 69, B: 39, A: 255}, color.RGBA{R: 38, G: 103, B: 118, A: 255}, color.RGBA{R: 173, G: 210, B: 224, A: 255}}),
			)

			var imgPng bytes.Buffer

			// Draws image
			img := w.Draw()
			// Encodes it
			_ = png.Encode(&imgPng, img)

			// Sends it
			<-c
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Files: []*discordgo.File{
					{
						Name:        "wordcloud.png",
						ContentType: "image/png",
						Reader:      &imgPng,
					},
				},
			})
		},

		"undelete": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var (
				number      uint
				m           discordgo.Message
				messageJSON []byte
				toSend      string
				toAdd       string
			)

			if len(i.ApplicationCommandData().Options) > 0 {
				number = uint(i.ApplicationCommandData().Options[0].UintValue())
			} else {
				// Default value
				number = 3
			}

			toSend = "Last " + strconv.Itoa(int(number)) + " deleted messages:\n```"

			rows, err := db.Query("SELECT message FROM messages WHERE deleted = 1 AND channelID=? ORDER BY JSON_VALUE(message, '$.timestamp') DESC LIMIT ?", i.ChannelID, number)
			if err != nil {
				lit.Error("Can't query database, %s", err)
				return
			}

			for rows.Next() {
				toAdd = ""

				err = rows.Scan(&messageJSON)
				if err != nil {
					lit.Error("Can't scan m, %s", err)
					continue
				}

				err = json.Unmarshal(messageJSON, &m)
				if err != nil {
					lit.Error("Can't unmarshal JSON, %s", err)
					continue
				}

				for _, a := range m.Attachments {
					toAdd += a.ID + "\n"
				}

				for _, e := range m.Embeds {
					for _, f := range e.Fields {
						toAdd += f.Name + ": " + f.Value + "\n"
					}
				}

				if m.Author != nil {
					toSend += m.Author.Username
				}
				toSend += ": " + toAdd + m.Content + "\n"
			}

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Undelete", strings.TrimSuffix(toSend, "\n")+"```").
				SetColor(0x7289DA).MessageEmbed, i.Interaction, c)
		},

		"markov": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			tokens := []string{gomarkov.StartToken}
			for tokens[len(tokens)-1] != gomarkov.EndToken {
				next, _ := server[i.GuildID].model.Generate(tokens[(len(tokens) - 1):])
				tokens = append(tokens, next)
			}

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Markov", strings.Join(tokens[1:len(tokens)-1], " ")).
				SetColor(0x7289DA).MessageEmbed, i.Interaction, c)
		},

		"longest": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var (
				rows                 *sql.Rows
				data                 []byte
				channelID, messageID string
				m                    LightMessage
				cont                 = 1
				highest              int
				err                  error
				embed                = NewEmbed().SetTitle(s.State.User.Username).SetColor(0x7289DA)
			)

			if len(i.ApplicationCommandData().Options) > 0 {
				rows, err = db.Query("SELECT message, channelID, messageID FROM messages WHERE channelID=? AND deleted=0 ORDER BY length(JSON_VALUE(message, '$.content')) DESC LIMIT 10", i.ApplicationCommandData().Options[0].ChannelValue(nil).ID)
			} else {
				rows, err = db.Query("SELECT message, channelID, messageID FROM messages WHERE guildID=? AND deleted=0 ORDER BY length(JSON_VALUE(message, '$.content')) DESC LIMIT 10", i.GuildID)
			}

			if err != nil {
				lit.Error("Error querying database: %s", err.Error())
				return
			}

			for rows.Next() {
				err = rows.Scan(&data, &channelID, &messageID)
				if err == nil {
					_ = json.Unmarshal(data, &m)

					if len(m.Content) < 100 {
						highest = len(m.Content)
					} else {
						highest = 100
					}

					embed.AddField(strconv.Itoa(cont), fmt.Sprintf("[%s](https://discord.com/channels/%s/%s/%s) - by %s\n", m.Content[0:highest], i.GuildID, channelID, messageID, m.Author.Username))
					cont++
				}
			}

			sendEmbedInteraction(s, embed.MessageEmbed, i.Interaction, c)
		},
		"poll": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var n int
			question := i.ApplicationCommandData().Options[0].StringValue()
			group := i.ApplicationCommandData().Options[1].StringValue()

			// If the group doesn't exist, just quit
			_ = db.QueryRow("SELECT COUNT(*) FROM pollsGroup WHERE serverID=? AND name=?", i.GuildID, group).Scan(&n)
			if n == 0 {
				sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "Group not found!").
					SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
				return
			}

			// Create the poll message
			msg, _ := s.ChannelMessageSendEmbed(i.ChannelID, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", question).MessageEmbed)

			// Adds the message to the db
			_, _ = db.Exec("INSERT INTO pollState (messageID, question, groupName, guildID) VALUES (?, ?, ?, ?)", msg.ID, question, group, i.GuildID)

			// And the map
			server[i.GuildID].polls[msg.ID] = true

			// Add the reactions
			_ = s.MessageReactionAdd(msg.ChannelID, msg.ID, "👍")
			_ = s.MessageReactionAdd(msg.ChannelID, msg.ID, "👎")
		},

		"creategroup": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var n int
			_ = db.QueryRow("SELECT COUNT(*) FROM pollsGroup WHERE serverID=? AND name=?", i.GuildID, i.ApplicationCommandData().Options[0].StringValue()).Scan(&n)
			if n == 0 {
				_, _ = db.Exec("INSERT INTO pollsGroup (serverID, name, createdBy) VALUES (?, ?, ?)", i.GuildID, i.ApplicationCommandData().Options[0].StringValue(), i.Member.User.ID)
				sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "Group created!").
					SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
			} else {
				sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "Group already exists!").
					SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
			}
		},

		"deletegroup": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var userID string
			_ = db.QueryRow("SELECT createdBy FROM pollsGroup WHERE serverID=? AND name=?", i.GuildID, i.ApplicationCommandData().Options[0].StringValue()).Scan(&userID)

			if userID == "" {
				sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "Group not found!").
					SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
			}

			if userID == i.Member.User.ID {
				_, _ = db.Exec("DELETE FROM pollsGroup WHERE serverID=? AND name=?", i.GuildID, i.ApplicationCommandData().Options[0].StringValue())
				sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "Group deleted!").
					SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
			} else {
				sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "You are not the owner of this group!").
					SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
			}
		},

		// Adds a member to a group. Only the creator of the group can add people
		"addmember": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var createdBy, userIDs string
			_ = db.QueryRow("SELECT createdBy, userIDs FROM pollsGroup WHERE serverID=? AND name=?", i.GuildID, i.ApplicationCommandData().Options[0].StringValue()).Scan(&createdBy, &userIDs)

			if createdBy == "" {
				sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "Group not found!").
					SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
			}

			if createdBy == i.Member.User.ID {
				user := i.ApplicationCommandData().Options[1].UserValue(s)

				// If the user is already in the group, just return
				if strings.Contains(userIDs, user.ID) {
					sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "User already in the group!").
						SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
					return
				}

				// Gets the old members, and adds the new one
				if userIDs == "" {
					userIDs = user.ID
				} else {
					userIDs += "," + user.ID
				}

				_, _ = db.Exec("UPDATE pollsGroup SET userIDs=? WHERE serverID=? AND name=?", userIDs, i.GuildID, i.ApplicationCommandData().Options[0].StringValue())

				// Adds the nickname to the database
				_, _ = db.Exec("INSERT IGNORE INTO users (id, nickname) VALUES (?, ?)", user.ID, user.Username)

				sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "Member added!").MessageEmbed, i.Interaction, time.Second*3, c)
			} else {
				sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "You are not the owner of this group!").
					SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
			}
		},

		// Removes a member from a group. Only the creator of the group can remove people
		"removemember": func(s *discordgo.Session, i *discordgo.InteractionCreate, c chan struct{}) {
			var createdBy, userIDs string
			_ = db.QueryRow("SELECT createdBy, userIDs FROM pollsGroup WHERE serverID=? AND name=?", i.GuildID, i.ApplicationCommandData().Options[0].StringValue()).Scan(&createdBy, &userIDs)

			if createdBy == "" {
				sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "Group not found!").
					SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
			}

			if createdBy == i.Member.User.ID {
				// If the user is not in the group, just return
				if !strings.Contains(userIDs, i.ApplicationCommandData().Options[1].UserValue(s).ID) {
					sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "User not in the group!").
						SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
					return
				}

				// Gets the old members, and adds the new one
				userIDs = strings.Replace(userIDs, ","+i.ApplicationCommandData().Options[1].UserValue(nil).ID, "", -1)

				_, _ = db.Exec("UPDATE pollsGroup SET userIDs=? WHERE serverID=? AND name=?", userIDs, i.GuildID, i.ApplicationCommandData().Options[0].StringValue())

			} else {
				sendAndDeleteEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Poll", "You are not the owner of this group!").
					SetColor(0x7289DA).MessageEmbed, i.Interaction, time.Second*3, c)
			}
		},
	}
)
