package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

// Encodes the video for audio playback
func (v *VoiceInstance) DCA(path string, isMpeg bool) {
	log.Printf("INFO: Starting DCA function with path: %s", path)

	// Log nowPlaying info which should have been set before this call
	if v.nowPlaying.Title != "" {
		log.Printf("INFO: Streaming audio for: %s", v.nowPlaying.Title)
	}

	var audioPath string
	var originalURL string

	// Determine audio path based on input type
	if isMpeg {
		// Local files in the mpegs directory
		audioPath = "mpegs/" + path
		log.Printf("INFO: Using local file: %s", audioPath)
	} else if strings.HasPrefix(path, "downloads/") || strings.HasPrefix(path, "./downloads/") {
		// Direct paths to files in the downloads directory
		audioPath = path
		log.Printf("INFO: Using direct file path: %s", audioPath)
	} else if strings.HasPrefix(path, "http") {
		// For YouTube URLs
		log.Printf("INFO: Processing URL: %s", path)

		// Extract video ID and construct original YouTube URL if needed
		var videoID string
		if strings.Contains(path, "youtube.com/watch?v=") {
			parts := strings.Split(path, "v=")
			if len(parts) > 1 {
				videoID = strings.Split(parts[1], "&")[0]
				originalURL = path
			}
		} else if strings.Contains(path, "youtu.be/") {
			parts := strings.Split(path, "youtu.be/")
			if len(parts) > 1 {
				videoID = strings.Split(parts[1], "?")[0]
				originalURL = "https://www.youtube.com/watch?v=" + videoID
			}
		} else if strings.Contains(path, "videoplayback") && strings.Contains(path, "id=") {
			// Extract ID from videoplayback URL and construct original YouTube URL
			parts := strings.Split(path, "id=")
			if len(parts) > 1 {
				videoID = strings.Split(parts[1], "&")[0]
				originalURL = "https://www.youtube.com/watch?v=" + videoID
			}
		}

		if videoID == "" {
			log.Printf("ERROR: Could not extract video ID from URL")
			return
		}
		log.Printf("INFO: Extracted video ID: %s", videoID)

		// Create downloads directory if it doesn't exist
		downloadDir := "downloads"
		if err := os.MkdirAll(downloadDir, 0755); err != nil {
			log.Printf("ERROR: Failed to create downloads directory: %v", err)
			return
		}

		// Define MP3 path
		mp3Path := filepath.Join(downloadDir, videoID+".mp3")

		// Check if MP3 already exists
		if _, err := os.Stat(mp3Path); err == nil {
			log.Printf("INFO: Using cached MP3 file: %s", mp3Path)
			audioPath = mp3Path
		} else {
			log.Printf("INFO: Downloading audio from YouTube: %s", originalURL)

			// Set up environment with YouTube token
			env := os.Environ()
			env = append(env, "YT_TOKEN="+os.Getenv("YT_TOKEN"))

			// Use yt-dlp to download audio in MP3 format
			cmd := exec.Command("yt-dlp",
				"--no-playlist",         // Don't download playlists
				"-x",                    // Extract audio
				"--audio-format", "mp3", // Convert to MP3
				"--audio-quality", "192K", // Set quality
				"--no-warnings", // Reduce noise in logs
				"--progress",    // Show progress
				"-o", mp3Path,   // Output file
				originalURL) // Original YouTube URL

			cmd.Env = env
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("ERROR: Failed to download audio: %v", err)
				log.Printf("yt-dlp output: %s", string(output))
				// Clean up partial file if it exists
				os.Remove(mp3Path)
				return
			}

			// Verify the MP3 file exists and has content
			if info, err := os.Stat(mp3Path); err != nil || info.Size() == 0 {
				log.Printf("ERROR: MP3 file is missing or empty")
				if err == nil {
					os.Remove(mp3Path)
				}
				return
			}

			log.Printf("INFO: Successfully downloaded audio to MP3: %s", mp3Path)
			audioPath = mp3Path
		}
	} else {
		log.Printf("ERROR: Unsupported path format: %s", path)
		return
	}

	// Verify file exists and get size
	fileInfo, err := os.Stat(audioPath)
	if err != nil {
		log.Printf("ERROR: Audio file does not exist or cannot be accessed: %s", audioPath)
		return
	}
	log.Printf("INFO: Audio file size: %d bytes", fileInfo.Size())

	// Find voice channel before attempting to join
	voiceChannelID, err := v.findUserVoiceChannel()
	if err != nil || voiceChannelID == "" {
		log.Printf("ERROR: Failed to find a voice channel: %v", err)
		return
	}

	// Play the MP3 file using the direct method
	log.Printf("INFO: Playing MP3 file using direct method: %s", audioPath)
	playMP3Direct(v.session, v.guildID, voiceChannelID, audioPath)
}

// Helper function to find the user's voice channel
func (v *VoiceInstance) findUserVoiceChannel() (string, error) {
	// Get guild information
	g, err := v.session.State.Guild(v.guildID)
	if err != nil {
		return "", err
	}

	// First, try to find any active voice states in the guild (users in voice channels)
	if len(g.VoiceStates) > 0 {
		// Just use the first voice channel we find with users in it
		return g.VoiceStates[0].ChannelID, nil
	}

	// If no users are in voice channels, find a voice channel in the guild
	channels, err := v.session.GuildChannels(v.guildID)
	if err != nil {
		return "", err
	}

	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildVoice {
			return channel.ID, nil
		}
	}

	return "", nil
}

// DirectDCA directly plays an audio file from the provided path
func (v *VoiceInstance) DirectDCA(filePath string) {
	log.Printf("INFO: Starting DirectDCA function with file: %s", filePath)

	// Basic check for file existence
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("ERROR: Audio file does not exist: %s", filePath)
		return
	}

	// Verify the audio file before encoding
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("ERROR: Cannot access audio file: %v", err)
		return
	}
	log.Printf("INFO: Audio file size: %d bytes", fileInfo.Size())

	// DCA encoding options for better stability
	options := dca.StdEncodeOptions
	options.RawOutput = false
	options.Volume = 256  // Normal volume
	options.Bitrate = 128 // Good balance of quality and bandwidth
	options.Application = dca.AudioApplicationAudio
	options.PacketLoss = 1       // Compensate for potential packet loss
	options.BufferedFrames = 200 // Larger buffer for stability
	options.FrameDuration = 20   // 20ms frame duration (standard)
	options.Threads = 4          // Use more threads for encoding
	options.VBR = true           // Variable bitrate for better quality

	log.Printf("DEBUG: Using audio settings - Bitrate: %d, Volume: %d, App: %s, Buffer: %d frames",
		options.Bitrate, options.Volume, options.Application, options.BufferedFrames)

	// Ensure voice connection is ready
	if v.voice == nil || !v.voice.Ready {
		log.Printf("ERROR: Voice connection is not ready")
		return
	}

	// Start speaking before creating the encoding session
	log.Printf("INFO: Setting speaking state to true")
	err = v.voice.Speaking(true)
	if err != nil {
		log.Printf("ERROR: Failed to set speaking state: %v", err)
		return
	}

	// Give Discord a moment to register the speaking state
	time.Sleep(500 * time.Millisecond)

	// Encode the file
	log.Printf("INFO: Creating encoding session for %s", filePath)
	encodingSession, err := dca.EncodeFile(filePath, options)
	if err != nil {
		log.Printf("ERROR: Failed to create encoding session: %v", err)
		v.voice.Speaking(false)
		return
	}
	defer encodingSession.Cleanup()
	v.encoder = encodingSession

	// Create a stream
	log.Printf("INFO: Creating audio stream")
	done := make(chan error)
	stream := dca.NewStream(encodingSession, v.voice, done)
	if stream == nil {
		log.Printf("ERROR: Failed to create stream")
		v.voice.Speaking(false)
		return
	}
	v.stream = stream

	// Playback duration estimation based on file size and bitrate
	estimatedDuration := float64(fileInfo.Size()) / (float64(options.Bitrate) * 1000 / 8)
	log.Printf("INFO: Streaming audio (should play for approximately %.0f seconds)", estimatedDuration)

	// Set up a heartbeat to maintain the speaking state
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Set up a timer to check for premature stream end
	earlyEndTimer := time.NewTimer(2 * time.Second)
	defer earlyEndTimer.Stop()

	// Stream monitoring
	streamStartTime := time.Now()
	keepAlive := true

	// Create a goroutine to periodically update the speaking state
	go func() {
		for keepAlive {
			select {
			case <-ticker.C:
				if v.voice != nil && v.voice.Ready {
					// Refresh speaking state to keep connection alive
					v.voice.Speaking(true)
					log.Printf("DEBUG: Refreshed speaking state at %.2f seconds", time.Since(streamStartTime).Seconds())
				} else {
					return
				}
			}
		}
	}()

	// Wait for stream to finish or early end check
	select {
	case <-earlyEndTimer.C:
		// If we reach here, the stream has been running for at least 2 seconds,
		// which is a good sign, so continue waiting for the full stream
		log.Printf("DEBUG: Stream running for 2+ seconds, continuing to monitor")

		// Now wait for the actual stream to complete
		streamErr := <-done
		keepAlive = false
		duration := time.Since(streamStartTime)

		log.Printf("INFO: Stream lasted for %.2f seconds", duration.Seconds())
		if streamErr != nil && streamErr != io.EOF {
			log.Printf("ERROR: DCA stream error: %v", streamErr)
		} else {
			log.Printf("INFO: DCA stream completed successfully")
		}

	case streamErr := <-done:
		// Stream ended before the early end check
		keepAlive = false
		duration := time.Since(streamStartTime)

		log.Printf("INFO: Stream lasted for %.2f seconds", duration.Seconds())
		if streamErr != nil && streamErr != io.EOF {
			log.Printf("ERROR: DCA stream error: %v", streamErr)
		} else {
			log.Printf("INFO: DCA stream completed successfully")
		}

		// If the stream ended too quickly, it might indicate an issue
		if duration.Seconds() < 1.0 && estimatedDuration > 5.0 {
			log.Printf("WARNING: Audio stream ended too quickly (%.2f seconds). File may be corrupted or format incompatible", duration.Seconds())
			log.Printf("INFO: Attempting alternative playback method...")

			// Try playing the file directly through ffmpeg to Discord
			go v.playWithFFmpeg(filePath)
			return
		}
	}

	// Stop speaking
	if v.voice != nil {
		log.Printf("INFO: Setting speaking state to false")
		v.voice.Speaking(false)
	}
}

// playWithFFmpeg is a backup method that uses ffmpeg to send audio directly to Discord
func (v *VoiceInstance) playWithFFmpeg(filePath string) {
	if v.voice == nil || !v.voice.Ready {
		log.Printf("ERROR: Voice connection is not ready for ffmpeg playback")
		return
	}

	log.Printf("INFO: Attempting ffmpeg direct playback for %s", filePath)

	// Set speaking state
	err := v.voice.Speaking(true)
	if err != nil {
		log.Printf("ERROR: Failed to set speaking state for ffmpeg playback: %v", err)
		return
	}

	// Create a temporary DCA file for Discord compatibility
	tempFile := filePath + ".dca"

	// Convert MP3 to DCA format using ffmpeg pipe to dca-rs (stored in temp file)
	log.Printf("INFO: Converting MP3 to DCA format for compatibility")

	// Standard encoding options
	options := dca.StdEncodeOptions
	options.RawOutput = true // Set to true for direct file output
	options.Volume = 256
	options.Bitrate = 64 // Lower bitrate for better stability
	options.Application = dca.AudioApplicationAudio

	// Create the encoding session
	encodingSession, err := dca.EncodeFile(filePath, options)
	if err != nil {
		log.Printf("ERROR: Failed to create direct encoding session: %v", err)
		v.voice.Speaking(false)
		return
	}
	defer encodingSession.Cleanup()

	// Create output file for DCA data
	output, err := os.Create(tempFile)
	if err != nil {
		log.Printf("ERROR: Failed to create temp DCA file: %v", err)
		v.voice.Speaking(false)
		return
	}
	defer output.Close()
	defer os.Remove(tempFile) // Clean up temp file when done

	// Copy the encoded data to the temp file
	_, err = io.Copy(output, encodingSession)
	if err != nil {
		log.Printf("ERROR: Failed to write DCA data: %v", err)
		v.voice.Speaking(false)
		return
	}

	// Close the output file and reopen it for reading
	output.Close()

	// Open the DCA file for reading
	dcaFile, err := os.Open(tempFile)
	if err != nil {
		log.Printf("ERROR: Failed to open DCA file for playback: %v", err)
		v.voice.Speaking(false)
		return
	}
	defer dcaFile.Close()

	// Create a new stream reader
	decoder := dca.NewDecoder(dcaFile)

	// Start a ticker for status updates and keepalive
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Stream start time
	streamStartTime := time.Now()

	// Flag to track if stream is active
	streamActive := true

	// Start keepalive goroutine
	go func() {
		for streamActive {
			select {
			case <-ticker.C:
				if v.voice != nil && v.voice.Ready {
					// Refresh speaking state
					v.voice.Speaking(true)
					log.Printf("DEBUG: DCA file playback running for %.2f seconds", time.Since(streamStartTime).Seconds())
				} else {
					return
				}
			}
		}
	}()

	// Read and send frames
	frameCount := 0
	for {
		frame, err := decoder.OpusFrame()
		if err != nil {
			if err != io.EOF {
				log.Printf("ERROR: Error decoding opus frame: %v", err)
			} else {
				log.Printf("INFO: Reached end of DCA file")
			}
			break
		}

		// Send the frame
		select {
		case v.voice.OpusSend <- frame:
			frameCount++
			// Sleep for frame duration (20ms) to prevent flooding
			time.Sleep(20 * time.Millisecond)
		case <-time.After(1 * time.Second):
			log.Printf("ERROR: Timeout sending opus frame to Discord")
			break
		}
	}

	// Cleanup
	streamActive = false
	duration := time.Since(streamStartTime)

	if v.voice != nil {
		log.Printf("INFO: Setting speaking state to false")
		v.voice.Speaking(false)
	}

	log.Printf("INFO: DCA file playback completed after %.2f seconds, sent %d frames",
		duration.Seconds(), frameCount)
}
