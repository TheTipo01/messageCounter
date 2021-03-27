package main

import (
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	jsoniter "github.com/json-iterator/go"
)

var (
	// Commands
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "stats",
			Description: "Prints stats for a given channel, or if not specified for the entire server.",
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
		// Prints stats for a given channel, or if not specified for the entire server.
		"stats": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var (
				mex                 *sql.Rows
				err                 error
				json                = jsoniter.ConfigFastest
				messageJSON, toSend string
				m                   discordgo.Message
				cont, authorNil     int
				words               = make(map[string]int)
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
					words[m.Author.ID] += wordCount(m.Content)
					messages[m.Author.ID]++
				} else {
					authorNil++
				}
			}

			// Characters
			toSend = ""
			chr := sorting(characters)
			for i, kv := range chr {
				cont += kv.Value
				toSend += fmt.Sprintf("%d) %s: %d\n", i+1, people[kv.Key], kv.Value)
				charPerMex[kv.Key] = kv.Value / messages[kv.Key]
			}
			toSend = fmt.Sprintf("Number of characters sent: %d\n\n", cont) + toSend

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Characters", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)

			// Characters per message
			toSend = ""
			cont = 0
			for i, kv := range sorting(charPerMex) {
				cont += kv.Value
				toSend += fmt.Sprintf("%d) %s: %d\n", i+1, people[kv.Key], kv.Value)
			}
			toSend = fmt.Sprintf("Number of characters per message sent: %d\n\n", cont) + toSend

			sendEmbedInteractionFollowup(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Characters per message", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)

			// Words
			toSend = ""
			cont = 0
			for i, kv := range sorting(words) {
				cont += kv.Value
				toSend += fmt.Sprintf("%d) %s: %d\n", i+1, people[kv.Key], kv.Value)
			}
			toSend = fmt.Sprintf("Number of words: %d\n\n", cont) + toSend

			sendEmbedInteractionFollowup(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Words", toSend).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)
		},
	}
)
