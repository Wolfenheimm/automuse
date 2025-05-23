package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"bufio"
	"encoding/binary"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"layeh.com/gopus"
)

// Get & queue audio in a YouTube video / playlist
func queueSong(m *discordgo.MessageCreate) {
	commData, commDataIsValid := sanitizeQueueSongInputs(m)
	queueLenBefore := len(queue)

	if commDataIsValid {
		// Check if a youtube link is present
		if strings.Contains(m.Content, "https://www.youtube") {
			// Check if the link is a playlist or a simple video
			if strings.Contains(m.Content, "list") && strings.Contains(m.Content, "-pl") || strings.Contains(m.Content, "/playlist?") {
				prepPlaylistCommand(commData, m)
			} else if strings.Contains(m.Content, "watch") && !strings.Contains(m.Content, "-pl") {
				prepWatchCommand(commData, m)
			}
			resetSearch() // In case a search was called prior to this
		} else {
			// Search or queue input was sent
			prepSearchQueueSelector(commData, m)
		}

		// If there's nothing playing and the queue grew
		if v.nowPlaying == (Song{}) && len(queue) >= 1 {
			joinVoiceChannel()
			prepFirstSongEntered(m, false)
		} else if !searchRequested {
			prepDisplayQueue(commData, queueLenBefore, m)
		}

		commDataIsValid = false
	}
}

// Hidden play command, used for testing purposes
func queueKudasai(m *discordgo.MessageCreate) {
	commData := []string{"queue", "https://www.youtube.com/watch?v=35AgDDPQE48"}
	prepWatchCommand(commData, m)
}

// Queue a list of songs
func queueStuff(m *discordgo.MessageCreate) {
	files, err := os.ReadDir("mpegs/")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		song = fillSongInfo(m.ChannelID, m.Author.ID, m.ID, "manual entry", file.Name(), "none")
		queue = append(queue, song)
	}

	if v.nowPlaying == (Song{}) && len(queue) >= 1 {
		joinVoiceChannel()
		prepFirstSongEntered(m, true)
	}
}

// Stops current song and empties the queue
func stop(m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Stopping ["+v.nowPlaying.Title+"] & Clearing Queue :octagonal_sign:")
	v.stop = true
	queue = []Song{}
	resetSearch()

	if v.encoder != nil {
		v.encoder.Cleanup()
	}

	if v.voice != nil {
		v.voice.Disconnect()
	}
}

// Skips the current song
func skip(m *discordgo.MessageCreate) {
	// Check if skipping current song or skipping to another song
	if m.Content == "skip" {
		if v.nowPlaying == (Song{}) {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Queue is empty - There's nothing to skip!")
		} else {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Skipping "+v.nowPlaying.Title+" :loop:")
			prepSkip()
			resetSearch()
			log.Println("Skipped " + v.nowPlaying.Title)
		}
	} else if strings.Contains(m.Content, "skip to ") {
		msgData := strings.Split(m.Content, " ")
		// Can only accept 3 params: skip to #
		if len(msgData) == 3 {
			// The third parameter must be a number
			if input, err := strconv.Atoi(msgData[2]); err == nil {
				// Ensure input is greater than 0 and less than the length of the queue
				if input <= len(queue) && input > 0 {
					var tmp []Song
					for i, value := range queue {
						if i >= input-1 {
							tmp = append(tmp, value)
						}
					}
					s.ChannelMessageSend(m.ChannelID, "**[Muse]** Jumping to "+queue[input-1].Title+" :leftwards_arrow_with_hook: ")
					log.Printf("Jumping to [%s]", queue[input-1])
					queue = tmp
					prepSkip()
					resetSearch()
				} else {
					s.ChannelMessageSend(m.ChannelID, "**[Muse]** Selected input was not in queue range")
				}
			}
		}
	}
}

// Fetches and displays the queue
func displayQueue(m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Fetching Queue...")
	queueList := ":musical_note:   QUEUE LIST   :musical_note:\n"
	if v.nowPlaying != (Song{}) {
		queueList = queueList + "Now Playing: " + v.nowPlaying.Title + "  ->  Queued by <@" + v.nowPlaying.User + "> \n"
		for index, element := range queue {
			queueList = queueList + " " + strconv.Itoa(index+1) + ". " + element.Title + "  ->  Queued by <@" + element.User + "> \n"
			if index+1 == 14 {
				log.Println(queueList)
				s.ChannelMessageSend(m.ChannelID, queueList)
				queueList = ""
			}
		}
		s.ChannelMessageSend(m.ChannelID, queueList)
		log.Println(queueList)
	} else {
		s.ChannelMessageSend(m.ChannelID, queueList)
	}
}

// Removes a song from the queue at a specific position
func remove(m *discordgo.MessageCreate) {
	// Split the message to get which song to remove from the queue
	commData := strings.Split(m.Content, " ")
	var msgToUser string
	if len(commData) == 2 {
		if queuePos, err := strconv.Atoi(commData[1]); err == nil {
			if queue != nil {
				if 1 <= queuePos && queuePos <= len(queue) {
					queuePos--
					var songTitle = queue[queuePos].Title
					var tmpQueue []Song
					tmpQueue = queue[:queuePos]
					tmpQueue = append(tmpQueue, queue[queuePos+1:]...)
					queue = tmpQueue
					msgToUser = fmt.Sprintf("**[Muse]** Removed %s.", songTitle)
				} else {
					msgToUser = "**[Muse]** The selection was out of range."
				}
			} else {
				msgToUser = "**[Muse]** There is no queue to remove songs from."
			}
		}
		s.ChannelMessageSend(m.ChannelID, msgToUser)
	}
}

// Plays a specific MP3 file (bo28mIVKyKg.mp3) from the downloads folder
func playSpecialFile(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Check if the user is in a voice channel
	userVoiceChannel := SearchVoiceChannel(m.Author.ID)
	if userVoiceChannel == "" {
		log.Println("User not in a voice channel")
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** You must be in a voice channel to use this command.")
		return
	}

	// Check if the file exists
	fileName := "bo28mIVKyKg.mp3"
	filePath := "downloads/" + fileName
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Println("File does not exist:", filePath)
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** The special audio file could not be found.")
		return
	}

	// Send message to user
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Playing special audio file 🎵")

	// Setup voice instance properly
	v.guildID = SearchGuild(m.ChannelID)
	v.session = s

	// Play MP3 directly using the new direct playback function
	log.Printf("INFO: Playing MP3 file directly: %s", filePath)
	playMP3Direct(s, v.guildID, userVoiceChannel, filePath)

	// Confirm completion
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Finished playing special audio 🎵")
}

// Direct MP3 playback using ffmpeg for better compatibility
func playMP3Direct(s *discordgo.Session, guildID, channelID, filePath string) {
	log.Printf("INFO: Starting direct MP3 playback: %s", filePath)

	// Join voice channel
	log.Printf("INFO: Joining voice channel: %s", channelID)
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		log.Printf("ERROR: Failed to join voice channel: %v", err)
		return
	}

	// Ensure we disconnect when done
	defer vc.Disconnect()

	// Wait for voice connection to be ready
	ready := false
	for i := 0; i < 5; i++ {
		if vc != nil && vc.Ready {
			ready = true
			log.Printf("INFO: Voice connection is ready after %d attempts", i+1)
			break
		}
		log.Printf("INFO: Waiting for voice connection to be ready (attempt %d/5)", i+1)
		time.Sleep(1 * time.Second)
	}

	if !ready {
		log.Printf("ERROR: Voice connection failed to become ready after 5 attempts")
		return
	}

	// Allow connection to stabilize
	log.Printf("INFO: Waiting for voice connection to stabilize")
	time.Sleep(2 * time.Second)

	// Start speaking
	log.Printf("INFO: Setting speaking state to true")
	err = vc.Speaking(true)
	if err != nil {
		log.Printf("ERROR: Failed to set speaking state: %v", err)
		return
	}

	// Convert MP3 file to PCM audio using ffmpeg
	cmd := exec.Command("ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", filePath,
		"-f", "s16le", // PCM signed 16-bit little-endian
		"-ar", "48000", // 48KHz sampling rate
		"-ac", "2", // 2 channels (stereo)
		"-af", "volume=1.5", // Increase volume
		"pipe:1")

	ffmpegout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("ERROR: Failed to create ffmpeg stdout pipe: %v", err)
		vc.Speaking(false)
		return
	}

	ffmpegbuf := bufio.NewReader(ffmpegout)
	err = cmd.Start()
	if err != nil {
		log.Printf("ERROR: Failed to start ffmpeg: %v", err)
		vc.Speaking(false)
		return
	}

	// Create a channel to signal the end of audio playback
	done := make(chan bool)

	// Send audio to Discord in a separate goroutine
	go func() {
		var opusEncoder *gopus.Encoder
		var err error

		// Create Opus encoder
		opusEncoder, err = gopus.NewEncoder(48000, 2, gopus.Audio)
		if err != nil {
			log.Printf("ERROR: Failed to create Opus encoder: %v", err)
			done <- true
			return
		}

		// Set the bitrate
		opusEncoder.SetBitrate(128000) // 128 kbps

		// Buffer for reading audio data
		audiobuf := make([]int16, 960*2) // 960 samples * 2 channels

		// Send audio data to Discord
		for {
			// Check if skip was called
			if v.stop {
				log.Printf("INFO: Skip detected during playMP3Direct audio sending, stopping")
				break
			}

			// Read audio data
			err = binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				log.Printf("INFO: End of audio file reached")
				break
			}
			if err != nil {
				log.Printf("ERROR: Error reading from ffmpeg: %v", err)
				break
			}

			// Encode audio to Opus
			opus, err := opusEncoder.Encode(audiobuf, 960, 960*2*2)
			if err != nil {
				log.Printf("ERROR: Error encoding to Opus: %v", err)
				break
			}

			// Send to Discord
			vc.OpusSend <- opus
		}

		// Signal that we're done
		done <- true
	}()

	// Set up a ticker for maintaining speaking state
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Main loop - wait for audio to finish or keep alive
	for {
		select {
		case <-ticker.C:
			// Check if skip was called
			if v.stop {
				log.Printf("INFO: Skip detected during playMP3Direct main loop, stopping")
				// Kill ffmpeg process
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
				// Set speaking to false
				if vc != nil && vc.Ready {
					vc.Speaking(false)
				}
				return
			}

			// Keep the speaking state alive
			if vc != nil && vc.Ready {
				vc.Speaking(true)
				log.Printf("DEBUG: Refreshed speaking state")
			}
		case <-done:
			// Audio playback is complete, clean up
			log.Printf("INFO: Audio playback completed")

			// Wait for ffmpeg to finish
			err = cmd.Wait()
			if err != nil {
				log.Printf("ERROR: FFMPEG exited with error: %v", err)
			}

			// Set speaking to false
			if vc != nil && vc.Ready {
				vc.Speaking(false)
			}

			return
		}
	}
}

// Tries to force a non-Newark voice region for the guild
func forceVoiceRegion(s *discordgo.Session, guildID string) {
	regions, err := s.VoiceRegions()
	if err != nil {
		log.Println("Failed to get voice regions:", err)
		return
	}

	// Try to avoid Newark and select one of these optimal regions
	preferredRegions := []string{"us-west", "us-central", "eu-west", "singapore", "brazil"}

	var selectedRegion string
	for _, preferredID := range preferredRegions {
		for _, region := range regions {
			if region.ID == preferredID {
				selectedRegion = region.ID
				log.Printf("Setting voice region to: %s", region.Name)
				break
			}
		}
		if selectedRegion != "" {
			break
		}
	}

	// If no preferred region found, use first non-Newark region
	if selectedRegion == "" {
		for _, region := range regions {
			if !strings.Contains(strings.ToLower(region.Name), "newark") {
				selectedRegion = region.ID
				log.Printf("Using fallback voice region: %s", region.Name)
				break
			}
		}
	}

	if selectedRegion != "" {
		_, err = s.GuildEdit(guildID, &discordgo.GuildParams{
			Region: selectedRegion,
		})
		if err != nil {
			log.Println("Failed to update guild voice region:", err)
		} else {
			log.Printf("Updated guild voice region to: %s", selectedRegion)
		}
	}
}

// Attempts to play audio through Discord
func attemptDiscordPlayback(s *discordgo.Session, guildID, channelID, filePath string) bool {
	// Setup voice instance properly
	v.guildID = guildID
	v.session = s

	// Clean up any existing voice connection
	if v.voice != nil {
		v.voice.Speaking(false)
		v.voice.Disconnect()
		v.voice = nil
		time.Sleep(1 * time.Second)
	}

	// Try to join voice channel with specific parameters
	// selfDeaf=false might help with connection stability
	var err error
	v.voice, err = s.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		log.Printf("Failed to join voice channel: %v", err)
		return false
	}

	// Wait for voice connection to be ready
	ready := false
	for i := 0; i < 5; i++ {
		if v.voice != nil && v.voice.Ready {
			ready = true
			log.Println("Voice connection is ready after", i+1, "attempts")
			break
		}
		log.Printf("Waiting for voice connection to be ready (attempt %d/5)", i+1)
		time.Sleep(1 * time.Second)
	}

	if !ready {
		log.Println("Voice connection failed to become ready")
		if v.voice != nil {
			v.voice.Disconnect()
			v.voice = nil
		}
		return false
	}

	// Wait a moment for the connection to stabilize
	time.Sleep(1 * time.Second)

	// Try to set speaking state
	err = v.voice.Speaking(true)
	if err != nil {
		log.Printf("Failed to set speaking state: %v", err)
		v.voice.Disconnect()
		v.voice = nil
		return false
	}

	// Play using DirectDCA for simplicity
	directPlayResult := make(chan bool, 1)

	// Create a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		// Simple encoding options
		options := dca.StdEncodeOptions
		options.RawOutput = false
		options.Volume = 256
		options.Bitrate = 96 // Lower bitrate for better stability
		options.Application = dca.AudioApplicationAudio

		// Create encoding session
		log.Println("Creating encoding session")
		encodingSession, err := dca.EncodeFile(filePath, options)
		if err != nil {
			log.Printf("Failed to create encoding session: %v", err)
			directPlayResult <- false
			return
		}
		defer encodingSession.Cleanup()

		// Create stream
		log.Println("Creating audio stream")
		done := make(chan error)
		stream := dca.NewStream(encodingSession, v.voice, done)
		if stream == nil {
			log.Println("Failed to create stream")
			directPlayResult <- false
			return
		}

		// Check for stream error after a brief moment
		// This will detect if the stream fails immediately
		time.Sleep(1 * time.Second)
		select {
		case err := <-done:
			if err != nil && err != io.EOF {
				log.Printf("Stream failed quickly: %v", err)
				directPlayResult <- false
			} else if err == io.EOF {
				// Stream completed very quickly, not normal
				log.Println("Stream completed too quickly")
				directPlayResult <- false
			} else {
				// No error yet, but we've only waited 1 second - keep streaming
				log.Println("Stream appears to be working")
				directPlayResult <- true
			}
		default:
			// No error received yet, stream appears to be working
			log.Println("Stream is running")
			directPlayResult <- true
		}
	}()

	// Wait for either the direct play result or timeout
	select {
	case result := <-directPlayResult:
		// Clean up
		if v.voice != nil {
			v.voice.Speaking(false)
			v.voice.Disconnect()
			v.voice = nil
		}
		return result
	case <-ctx.Done():
		if v.voice != nil {
			v.voice.Disconnect()
			v.voice = nil
		}
		return false
	}
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
