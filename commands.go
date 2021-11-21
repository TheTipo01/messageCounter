package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	"github.com/goccy/go-json"
	"github.com/psykhi/wordclouds"
	"image/color"
	"image/png"
	"regexp"
	"strconv"
	"strings"
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
	}

	// Handler
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		// Prints the number of characters sent for a given channel, or if not specified for the entire server
		"characters": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var (
				mex         *sql.Rows
				err         error
				messageJSON []byte
				toSend      string
				m           discordgo.Message
				cont        int
				characters  = make(map[string]int)
				people      = make(map[string]string)
			)

			// If there's a specified channel, use it in the query
			if len(i.ApplicationCommandData().Options) > 0 {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=? AND channelID=?", i.GuildID, i.ApplicationCommandData().Options[0].ChannelValue(s).ID)
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
				mex         *sql.Rows
				err         error
				messageJSON []byte
				toSend      string
				m           discordgo.Message
				cont        int
				words       = make(map[string]int)
				people      = make(map[string]string)
				// Match non-space character sequences.
				re = regexp.MustCompile(`[\S]+`)
			)

			// If there's a specified channel, use it in the query
			if len(i.ApplicationCommandData().Options) > 0 {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=? AND channelID=?", i.GuildID, i.ApplicationCommandData().Options[0].ChannelValue(s).ID)
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

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Words", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)
		},

		// Prints the number of messages sent for a given channel, or if not specified for the entire server
		"messages": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var (
				mex         *sql.Rows
				err         error
				messageJSON []byte
				toSend      string
				m           discordgo.Message
				cont        int
				people      = make(map[string]string)
				messages    = make(map[string]int)
			)

			// If there's a specified channel, use it in the query

			if len(i.ApplicationCommandData().Options) > 0 {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=? AND channelID=?", i.GuildID, i.ApplicationCommandData().Options[0].ChannelValue(s).ID)
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

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Messages", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)
		},

		// Prints stats for a given channel, or if not specified for the entire server.
		"charsPerMex": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var (
				mex         *sql.Rows
				err         error
				messageJSON []byte
				toSend      string
				m           discordgo.Message
				cont        int
				characters  = make(map[string]int)
				people      = make(map[string]string)
				messages    = make(map[string]int)
				charPerMex  = make(map[string]int)
			)

			// If there's a specified channel, use it in the query
			if len(i.ApplicationCommandData().Options) > 0 {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=? AND channelID=?", i.GuildID, i.ApplicationCommandData().Options[0].ChannelValue(s).ID)
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

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Characters per message", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)

		},

		// Generates a word cloud for a channel, or the entire server if omitted
		"wordcloud": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var (
				mex         *sql.Rows
				err         error
				messageJSON []byte
				m           discordgo.Message
				words       = make(map[string]int)
			)

			// If there's a specified channel, use it in the query
			if len(i.ApplicationCommandData().Options) > 0 {
				mex, err = db.Query("SELECT message FROM messages WHERE guildID=? AND channelID=?", i.GuildID, i.ApplicationCommandData().Options[0].ChannelValue(s).ID)
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

				mSplitted := strings.Fields(strings.ToLower(m.Content))
				for _, word := range mSplitted {
					if utf8.RuneCountInString(word) > 3 {
						words[word]++
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

			// Send it in a channel
			sentImg, err := s.ChannelFileSend(i.ChannelID, "wordcloud.png", &imgPng)
			if err != nil {
				lit.Error("Error while sending image " + err.Error())
				return
			}

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).SetColor(0x7289DA).SetImage(m.Attachments[0].URL).
				MessageEmbed, i.Interaction)

			err = s.ChannelMessageDelete(sentImg.ChannelID, sentImg.ID)
			if err != nil {
				lit.Error("Error while deleting sent image " + err.Error())
			}
		},

		"undelete": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

			rows, err := db.Query("SELECT message FROM messages WHERE guildID=? AND channelID=? LIMIT ?", i.GuildID, i.ChannelID, number)
			if err != nil {
				lit.Error("Can't query database, %s", err)
				return
			}

			for rows.Next() {
				toAdd = ""

				err = rows.Scan(&m)

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
					toAdd += e.Description + "\n"
				}

				toSend += m.Author.Username + ": " + toAdd + m.Content + "\n"
			}

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Undelete", strings.TrimSuffix(toSend, "\n")+"```").
				SetColor(0x7289DA).MessageEmbed, i.Interaction)
		},
	}
)
