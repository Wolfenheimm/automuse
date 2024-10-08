package main

import (
	"log"
	"os"
)

// Joins the voice channel designated by the environment variables
func joinVoiceChannel() {
	// It was a chosen decision to have hard-coded guild and channel IDs
	// - This is because the bot is only used in one server, and the voice channel is always the same
	// - If you want to use this bot in different channels, there is definitely a dynamic way to do this
	// Note: It might not be possible for the bot to join multiple channels at once, this hasn't been tested
	generalChan := os.Getenv("GENERAL_CHAT_ID") // original value for guild id = v.guildID
	guildID := os.Getenv("GUILD_ID")

	var err error
	v.voice, err = s.ChannelVoiceJoin(guildID, generalChan, true, true)

	if err != nil {
		if _, ok := s.VoiceConnections[guildID]; ok {
			v.voice = s.VoiceConnections[guildID]
			log.Println("ERROR: Guild ID: ", guildID)
			log.Println("ERROR: Channel: ", generalChan)
			log.Println("ERROR: Error to join in a voice channel: ", err)
			log.Println("error connecting:", err)
			return
		} else {
			log.Println("error connecting:", err)
			return
		}
	}

	v.voice.Speaking(false)
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
