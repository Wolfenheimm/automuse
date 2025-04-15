package main

import (
	"log"
	"os/exec"
)

// Helper function to play audio locally using ffplay
func playLocalAudio(filePath string) {
	log.Println("Playing audio locally with ffplay")
	cmd := exec.Command("ffplay", "-nodisp", "-autoexit", filePath)
	err := cmd.Start()
	if err != nil {
		log.Println("Failed to start ffplay:", err)
		return
	}

	// Wait for playback to complete
	log.Println("Local audio playback started")
	cmd.Wait()
	log.Println("Local audio playback finished")
}

// Enhanced version of playLocalAudio with more options
func playLocalAudioEnhanced(filePath string, wait bool) {
	log.Println("Playing enhanced audio locally with ffplay")

	// Use higher quality settings
	cmd := exec.Command("ffplay", "-nodisp", "-autoexit", "-af", "volume=1.5", filePath)
	err := cmd.Start()
	if err != nil {
		log.Println("Failed to start ffplay:", err)
		return
	}

	log.Println("Enhanced local audio playback started")

	if wait {
		cmd.Wait()
		log.Println("Enhanced local audio playback finished")
	} else {
		log.Println("Not waiting for enhanced audio playback to finish")
	}
}
