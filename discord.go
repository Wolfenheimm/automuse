package main

import (
	"fmt"
	"log"
)

// Server-agnostic voice channel joining - finds user's current voice channel
func joinVoiceChannel() {
	if v.currentUserID == "" {
		log.Printf("ERROR: No current user ID set for voice operations")
		return
	}

	// Find the user's current voice channel dynamically
	userVoiceChannelID := SearchVoiceChannel(v.currentUserID)
	if userVoiceChannelID == "" {
		log.Printf("ERROR: User %s is not in any voice channel in guild %s", v.currentUserID, v.guildID)
		return
	}

	log.Printf("INFO: Found user %s in voice channel %s, attempting to join...", v.currentUserID, userVoiceChannelID)

	var err error
	v.voice, err = s.ChannelVoiceJoin(v.guildID, userVoiceChannelID, false, true)

	if err != nil {
		// Check if there's an existing connection we can reuse
		if existingConn, ok := s.VoiceConnections[v.guildID]; ok {
			v.voice = existingConn
			log.Printf("WARN: Reusing existing voice connection for guild %s", v.guildID)
		} else {
			log.Printf("ERROR: Failed to join voice channel - Guild ID: %s, Channel: %s, Error: %v",
				v.guildID, userVoiceChannelID, err)
			return
		}
	} else {
		log.Printf("INFO: Successfully joined voice channel %s in guild %s", userVoiceChannelID, v.guildID)
	}

	v.voice.Speaking(false)
}

// Enhanced version that accepts a specific user ID
func joinUserVoiceChannel(userID string) error {
	// Find the specific user's voice channel
	userVoiceChannelID := SearchVoiceChannel(userID)
	if userVoiceChannelID == "" {
		return fmt.Errorf("user %s is not in any voice channel in guild %s", userID, v.guildID)
	}

	log.Printf("INFO: Joining user %s's voice channel %s", userID, userVoiceChannelID)

	var err error
	v.voice, err = s.ChannelVoiceJoin(v.guildID, userVoiceChannelID, false, true)

	if err != nil {
		// Check if there's an existing connection we can reuse
		if existingConn, ok := s.VoiceConnections[v.guildID]; ok {
			v.voice = existingConn
			log.Printf("WARN: Reusing existing voice connection for guild %s", v.guildID)
			return nil
		} else {
			return fmt.Errorf("failed to join voice channel %s in guild %s: %v", userVoiceChannelID, v.guildID, err)
		}
	}

	v.voice.Speaking(false)
	log.Printf("INFO: Successfully joined voice channel %s in guild %s", userVoiceChannelID, v.guildID)
	return nil
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
