package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func joinVoiceChannel(m *discordgo.MessageCreate) {
	// Get the channel of the person who made the request
	authorChan := SearchVoiceChannel(m.Author.ID)

	// Join the channel of the person who made the request
	if authorChan != m.ChannelID {
		var err error
		v.voice, err = s.ChannelVoiceJoin(v.guildID, authorChan, true, true)

		if err != nil {
			if _, ok := s.VoiceConnections[v.guildID]; ok {
				v.voice = s.VoiceConnections[v.guildID]
			}
			log.Println("ERROR: Error to join in a voice channel: ", err)
		}

		v.voice.Speaking(false)
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** <@"+m.Author.ID+"> - I've joined your channel!")
	}
}

// Gets guild information
func SearchGuild(textChannelID string) (guildID string) {
	channel, _ := s.Channel(textChannelID)
	guildID = channel.GuildID
	return
}

// Searches the voice channel (used to look for the person who sent the message & what voice channel they're in)
func SearchVoiceChannel(user string) (voiceChannelID string) {
	for _, g := range s.State.Guilds {
		for _, v := range g.VoiceStates {
			if v.UserID == user {
				return v.ChannelID
			}
		}
	}
	return ""
}
