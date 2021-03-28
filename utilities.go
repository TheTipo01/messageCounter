package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	"sort"
)

// Sends embed as response to an interaction
func sendEmbedInteraction(s *discordgo.Session, embed *discordgo.MessageEmbed, i *discordgo.Interaction) {
	sliceEmbed := []*discordgo.MessageEmbed{embed}
	err := s.InteractionRespond(i, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionApplicationCommandResponseData{Embeds: sliceEmbed}})
	if err != nil {
		lit.Error("InteractionRespond failed: %s", err)
	}
}

func sendEmbedInteractionFollowup(s *discordgo.Session, embed *discordgo.MessageEmbed, i *discordgo.Interaction) {
	sliceEmbed := []*discordgo.MessageEmbed{embed}
	_, err := s.FollowupMessageCreate(s.State.User.ID, i, false, &discordgo.WebhookParams{Embeds: sliceEmbed})
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
