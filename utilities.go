package main

import (
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
)

var answers = []string{
	"It is certain",
	"Reply hazy, try again",
	"Don’t count on it",
	"It is decidedly so",
	"Ask again later",
	"My reply is no",
	"Without a doubt",
	"Better not tell you now",
	"My sources say no",
	"Yes definitely",
	"Cannot predict now",
	"Outlook not so good",
	"You may rely on it",
	"Concentrate and ask again",
	"Very doubtful",
	"As I see it, yes",
	"Most likely",
	"Outlook good",
	"Yes",
	"Signs point to yes",
}

// Sends embed as response to an interaction
func sendEmbedInteraction(s *discordgo.Session, embed *discordgo.MessageEmbed, i *discordgo.Interaction, c chan struct{}) {
	sliceEmbed := []*discordgo.MessageEmbed{embed}
	_, err := s.InteractionResponseEdit(i, &discordgo.WebhookEdit{Embeds: &sliceEmbed})
	if err != nil {
		lit.Error("InteractionRespond failed: %s", err)
	}
}

// Sends and delete after three second an embed in a given channel
func sendAndDeleteEmbedInteraction(s *discordgo.Session, embed *discordgo.MessageEmbed, i *discordgo.Interaction, wait time.Duration, c chan struct{}) {
	sendEmbedInteraction(s, embed, i, c)

	time.Sleep(wait)

	err := s.InteractionResponseDelete(i)
	if err != nil {
		lit.Error("InteractionResponseDelete failed: %s", err)
		return
	}
}

// Sorts a map into an array
func sorting(classifica map[string]int) []kv {
	var ss []kv
	for k, v := range classifica {
		ss = append(ss, kv{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	return ss
}
