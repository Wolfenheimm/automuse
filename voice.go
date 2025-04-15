package main

import (
	"fmt"
	"log"
	"time"
)

// JoinVoiceChannel joins a voice channel
func JoinVoiceChannel(guildID, channelID string) error {
	var err error
	maxRetries := 3
	retryDelay := time.Second * 2

	for i := 0; i < maxRetries; i++ {
		v.voice, err = s.ChannelVoiceJoin(guildID, channelID, false, true)
		if err == nil {
			log.Printf("INFO: Successfully joined voice channel %s in guild %s", channelID, guildID)
			return nil
		}

		log.Printf("WARN: Attempt %d/%d to join voice channel failed: %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("failed to join voice channel after %d attempts: %v", maxRetries, err)
}

// LeaveVoiceChannel leaves the current voice channel
func LeaveVoiceChannel() {
	if v.voice != nil {
		v.voice.Disconnect()
		v.voice = nil
		log.Printf("INFO: Left voice channel")
	}
} 