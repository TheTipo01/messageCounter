package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	"sort"
	"time"
)

// Sends embed as response to an interaction
func sendEmbedInteraction(s *discordgo.Session, embed *discordgo.MessageEmbed, i *discordgo.Interaction) {
	sliceEmbed := []*discordgo.MessageEmbed{embed}
	err := s.InteractionRespond(i, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: sliceEmbed}})
	if err != nil {
		lit.Error("InteractionRespond failed: %s", err)
	}
}

// Sends and delete after three second an embed in a given channel
func sendAndDeleteEmbedInteraction(s *discordgo.Session, embed *discordgo.MessageEmbed, i *discordgo.Interaction, wait time.Duration) {
	sendEmbedInteraction(s, embed, i)

	time.Sleep(wait)

	err := s.InteractionResponseDelete(i)
	if err != nil {
		lit.Error("InteractionResponseDelete failed: %s", err)
		return
	}
}

func sendEmbedInteractionFollowup(s *discordgo.Session, embed *discordgo.MessageEmbed, i *discordgo.Interaction) {
	sliceEmbed := []*discordgo.MessageEmbed{embed}
	_, err := s.FollowupMessageCreate(i, false, &discordgo.WebhookParams{Embeds: sliceEmbed})
	if err != nil {
		lit.Error("FollowupMessageCreate failed: %s", err)
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
