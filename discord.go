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

	// Validate required environment variables
	if guildID == "" {
		log.Printf("ERROR: GUILD_ID environment variable is not set")
		return
	}
	if generalChan == "" {
		log.Printf("ERROR: GENERAL_CHAT_ID environment variable is not set")
		return
	}

	var err error
	v.voice, err = s.ChannelVoiceJoin(guildID, generalChan, false, true) // Changed first param to false (mute=false)

	if err != nil {
		// Check if there's an existing connection we can reuse
		if existingConn, ok := s.VoiceConnections[guildID]; ok {
			v.voice = existingConn
			log.Printf("WARN: Reusing existing voice connection for guild %s", guildID)
		} else {
			log.Printf("ERROR: Failed to join voice channel - Guild ID: %s, Channel: %s, Error: %v",
				guildID, generalChan, err)
			return
		}
	} else {
		log.Printf("INFO: Successfully joined voice channel %s in guild %s", generalChan, guildID)
	}

	v.voice.Speaking(false)
}

// Gets guild information with proper error handling
func SearchGuild(textChannelID string) (guildID string) {
	if textChannelID == "" {
		log.Printf("ERROR: SearchGuild called with empty textChannelID")
		return ""
	}

	channel, err := s.Channel(textChannelID)
	if err != nil {
		log.Printf("ERROR: Failed to get channel info for %s: %v", textChannelID, err)
		return ""
	}

	if channel == nil {
		log.Printf("ERROR: Channel is nil for ID %s", textChannelID)
		return ""
	}

	guildID = channel.GuildID
	log.Printf("DEBUG: Found guild ID %s for channel %s", guildID, textChannelID)
	return
}

// Searches the voice channel (used to look for the person who sent the message & what voice channel they're in)
func SearchVoiceChannel(user string) (voiceChannelID string) {
	if user == "" {
		log.Printf("ERROR: SearchVoiceChannel called with empty user ID")
		return ""
	}

	if s == nil || s.State == nil {
		log.Printf("ERROR: Discord session or state is nil")
		return ""
	}

	for _, g := range s.State.Guilds {
		if g == nil {
			continue
		}
		for _, v := range g.VoiceStates {
			if v != nil && v.UserID == user {
				log.Printf("DEBUG: Found user %s in voice channel %s", user, v.ChannelID)
				return v.ChannelID
			}
		}
	}

	log.Printf("DEBUG: User %s not found in any voice channel", user)
	return ""
}
