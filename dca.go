package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"layeh.com/gopus"
)

// DCA Audio Constants
const (
	// Connection timeouts
	VoiceConnectionTimeout = 5 * time.Second
	VoiceReadyRetries      = 5
	VoiceReadyWaitTime     = 1 * time.Second

	// Audio processing timeouts
	AudioFrameTimeout  = 1 * time.Second
	SpeakingStateDelay = 500 * time.Millisecond
	SkipCheckInterval  = 100 * time.Millisecond
	HeartbeatInterval  = 5 * time.Second

	// Audio quality settings
	FFmpegBitrate     = 128000 // 128 kbps for Opus encoding
	FFmpegSampleRate  = 48000  // 48kHz - Discord standard
	FFmpegChannels    = 2      // Stereo
	OpusFrameSize     = 960    // Standard Opus frame size
	OpusFrameDuration = 20     // 20ms frame duration

	// Early stream detection
	MinStreamDuration   = 1 * time.Second // Minimum expected stream duration
	EarlyEndCheckDelay  = 2 * time.Second // Check for early stream end
	MinExpectedDuration = 5 * time.Second // Minimum duration for early end detection
)

// Encodes the video for audio playback
func (v *VoiceInstance) DCA(path string, isMpeg bool, useExistingConnection bool) {
	log.Printf("INFO: Starting DCA function with path: %s, useExistingConnection: %t", path, useExistingConnection)

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

		// Check metadata manager first for cached file
		if metadata, exists := metadataManager.GetSong(videoID); exists {
			log.Printf("INFO: Using cached MP3 file from metadata: %s", metadata.FilePath)
			audioPath = metadata.FilePath

			// Update usage statistics
			metadataManager.AddSong(videoID, metadata.Title, metadata.Duration, metadata.FilePath, metadata.FileSize)
		} else if _, err := os.Stat(mp3Path); err == nil {
			// File exists but not in metadata, add it
			log.Printf("INFO: Found existing MP3 file, adding to metadata: %s", mp3Path)
			audioPath = mp3Path

			if v.nowPlaying.Title != "" && v.nowPlaying.Duration != "" {
				if fileInfo, statErr := os.Stat(mp3Path); statErr == nil {
					metadataManager.AddSong(videoID, v.nowPlaying.Title, v.nowPlaying.Duration, mp3Path, fileInfo.Size())
				}
			}
		} else {
			log.Printf("INFO: Downloading audio from YouTube: %s", originalURL)

			// Set up environment with YouTube token
			env := os.Environ()
			env = append(env, "YT_TOKEN="+os.Getenv("YT_TOKEN"))

			// Use yt-dlp to download audio in MP3 format with comprehensive age restriction bypass
			var downloadErr error
			
			// Try different bypass methods in order of preference
			bypasses := [][]string{
				// Method 1: Basic age bypass
				{"--no-playlist", "-x", "--audio-format", "mp3", "--audio-quality", "256K", "--no-warnings", "--progress", "--age-limit", "99", "--no-check-certificate", "-o", mp3Path},
				// Method 2: With Chrome cookies
				{"--no-playlist", "-x", "--audio-format", "mp3", "--audio-quality", "256K", "--no-warnings", "--progress", "--age-limit", "99", "--no-check-certificate", "--cookies-from-browser", "chrome", "-o", mp3Path},
				// Method 3: With Safari cookies (macOS)
				{"--no-playlist", "-x", "--audio-format", "mp3", "--audio-quality", "256K", "--no-warnings", "--progress", "--age-limit", "99", "--no-check-certificate", "--cookies-from-browser", "safari", "-o", mp3Path},
				// Method 4: With Firefox cookies
				{"--no-playlist", "-x", "--audio-format", "mp3", "--audio-quality", "256K", "--no-warnings", "--progress", "--age-limit", "99", "--no-check-certificate", "--cookies-from-browser", "firefox", "-o", mp3Path},
			}
			
			for i, args := range bypasses {
				cmd := exec.Command("yt-dlp", append(args, originalURL)...)
				cmd.Env = env
				_, downloadErr = cmd.CombinedOutput()
				if downloadErr == nil {
					if i > 0 {
						log.Printf("INFO: yt-dlp download succeeded with bypass method %d (using browser cookies)", i+1)
					}
					break
				}
				log.Printf("DEBUG: yt-dlp download bypass method %d failed: %v", i+1, downloadErr)
				if i < len(bypasses)-1 {
					// Clean up partial file before next attempt
					os.Remove(mp3Path)
				}
			}
			
			if downloadErr != nil {
				log.Printf("ERROR: All yt-dlp download bypass methods failed: %v", downloadErr)
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

			// Add to metadata manager for future caching
			if v.nowPlaying.Title != "" && v.nowPlaying.Duration != "" {
				if fileInfo, statErr := os.Stat(mp3Path); statErr == nil {
					if err := metadataManager.AddSong(videoID, v.nowPlaying.Title, v.nowPlaying.Duration, mp3Path, fileInfo.Size()); err != nil {
						log.Printf("WARN: Failed to add song to metadata: %v", err)
					}
				}
			}
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

	var vc *discordgo.VoiceConnection

	if useExistingConnection {
		// Use existing voice connection
		if v.voice == nil || !v.voice.Ready {
			log.Printf("ERROR: No voice connection available for playback")
			return
		}
		vc = v.voice
		log.Printf("INFO: Using existing voice connection")
	} else {
		// Create new voice connection
		// Find voice channel before attempting to join
		voiceChannelID, err := v.findUserVoiceChannel()
		if err != nil || voiceChannelID == "" {
			log.Printf("ERROR: Failed to find a voice channel: %v", err)
			return
		}

		// Join voice channel
		log.Printf("INFO: Joining voice channel: %s", voiceChannelID)
		vc, err = v.session.ChannelVoiceJoin(v.guildID, voiceChannelID, false, false)
		if err != nil {
			log.Printf("ERROR: Failed to join voice channel: %v", err)
			return
		}
		defer vc.Disconnect()

		// Wait for voice connection to be ready
		ready := false
		for i := range VoiceReadyRetries {
			if vc != nil && vc.Ready {
				ready = true
				log.Printf("INFO: Voice connection is ready after %d attempts", i+1)
				break
			}
			log.Printf("INFO: Waiting for voice connection to be ready (attempt %d/%d)", i+1, VoiceReadyRetries)
			time.Sleep(VoiceReadyWaitTime)
		}

		if !ready {
			log.Printf("ERROR: Voice connection failed to become ready after %d attempts", VoiceReadyRetries)
			return
		}
	}

	// Play the MP3 file using the voice connection
	log.Printf("INFO: Playing MP3 file: %s", audioPath)
	playMP3WithExistingConnection(vc, audioPath)
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
	time.Sleep(SpeakingStateDelay)

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

	// Instead of using dca.NewStream, we'll manually handle frame sending to enable pause/resume
	go func() {
		defer func() { done <- nil }()

		for {
			// Check for stop condition
			if v.stop {
				log.Printf("INFO: Stop detected in DCA stream loop")
				return
			}

			// Read frame from encoder
			frame, err := encodingSession.OpusFrame()
			if err != nil {
				if err != io.EOF {
					log.Printf("ERROR: Error reading opus frame: %v", err)
				}
				return
			}

			// Handle pause - don't send frames to Discord
			if v.paused {
				// Sleep to maintain timing even when paused
				time.Sleep(20 * time.Millisecond)
				continue
			}

			// Normal operation - send frame to Discord
			select {
			case v.voice.OpusSend <- frame:
				// Frame sent successfully
			case <-time.After(1 * time.Second):
				log.Printf("ERROR: Timeout sending opus frame to Discord")
				return
			}

			// Maintain 20ms timing
			time.Sleep(20 * time.Millisecond)
		}
	}()

	v.stream = nil // We're not using dca.Stream anymore

	// Playback duration estimation based on file size and bitrate
	estimatedDuration := float64(fileInfo.Size()) / (float64(options.Bitrate) * 1000 / 8)
	log.Printf("INFO: Streaming audio (should play for approximately %.0f seconds)", estimatedDuration)

	// Set up a heartbeat to maintain the speaking state
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	// Set up a timer to check for premature stream end
	earlyEndTimer := time.NewTimer(EarlyEndCheckDelay)
	defer earlyEndTimer.Stop()

	// Stream monitoring
	streamStartTime := time.Now()
	keepAlive := true

	// Create a goroutine to periodically update the speaking state
	go func() {
		for keepAlive {
			<-ticker.C
			if v.voice != nil && v.voice.Ready {
				// Only refresh speaking state if not paused
				if !v.paused {
					v.voice.Speaking(true)
					log.Printf("DEBUG: Refreshed speaking state at %.2f seconds", time.Since(streamStartTime).Seconds())
				}
			} else {
				return
			}
		}
	}()

	// Wait for stream to finish or early end check
	select {
	case <-earlyEndTimer.C:
		// If we reach here, the stream has been running for at least 2 seconds,
		// which is a good sign, so continue waiting for the full stream
		log.Printf("DEBUG: Stream running for 2+ seconds, continuing to monitor")

		// Create a ticker to check for skip/pause during playback
		skipCheckTicker := time.NewTicker(SkipCheckInterval)
		defer skipCheckTicker.Stop()

		// Wait for either stream completion, skip command, or pause/resume
		for {
			select {
			case streamErr := <-done:
				keepAlive = false
				duration := time.Since(streamStartTime)

				log.Printf("INFO: Stream lasted for %.2f seconds", duration.Seconds())
				if streamErr != nil && streamErr != io.EOF {
					log.Printf("ERROR: DCA stream error: %v", streamErr)
				} else {
					log.Printf("INFO: DCA stream completed successfully")
				}
				goto cleanup

			case <-skipCheckTicker.C:
				// Check if skip was called
				if v.stop {
					log.Printf("INFO: Skip detected during playback, stopping stream")
					keepAlive = false

					// Stop the stream
					if v.stream != nil {
						// There's no direct stop method, so we'll cleanup the encoder
						// which should stop the stream
						if v.encoder != nil {
							v.encoder.Cleanup()
						}
					}
					goto cleanup
				}

				// Handle pause/resume logic
				if v.paused {
					// Song is paused, set speaking to false and stop sending audio
					if v.voice != nil && v.voice.Ready {
						v.voice.Speaking(false)
					}
					log.Printf("DEBUG: Song paused during playback - audio transmission will be handled in stream loop")

					// Wait for resume or skip
					for v.paused && !v.stop {
						time.Sleep(100 * time.Millisecond)
					}

					// If not stopped, resume
					if !v.stop && !v.paused {
						if v.voice != nil && v.voice.Ready {
							v.voice.Speaking(true)
						}
						log.Printf("DEBUG: Song resumed during playback - audio transmission will resume")
					}
				}
			}
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
		if duration < MinStreamDuration && estimatedDuration > MinExpectedDuration.Seconds() {
			log.Printf("WARNING: Audio stream ended too quickly (%.2f seconds). File may be corrupted or format incompatible", duration.Seconds())
			log.Printf("INFO: Attempting alternative playback method...")

			// Try playing the file directly through ffmpeg to Discord
			go v.playWithFFmpeg(filePath)
			return
		}
	}

cleanup:
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
			<-ticker.C
			if v.voice != nil && v.voice.Ready {
				// Refresh speaking state
				v.voice.Speaking(true)
				log.Printf("DEBUG: DCA file playback running for %.2f seconds", time.Since(streamStartTime).Seconds())
			} else {
				return
			}
		}
	}()

	// Read and send frames
	frameCount := 0
	for {
		// Check if skip was called
		if v.stop {
			log.Printf("INFO: Skip detected during ffmpeg playback, stopping")
			break
		}

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

// playMP3WithExistingConnection plays an MP3 file using an existing voice connection
// This prevents the bot from disconnecting and reconnecting between songs
func playMP3WithExistingConnection(vc *discordgo.VoiceConnection, filePath string) {
	log.Printf("INFO: Starting MP3 playback with existing connection: %s", filePath)

	// Verify voice connection is ready
	if vc == nil || !vc.Ready {
		log.Printf("ERROR: Voice connection is not ready for playback")
		return
	}

	// Start speaking
	log.Printf("INFO: Setting speaking state to true")
	err := vc.Speaking(true)
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
		"-ar", fmt.Sprintf("%d", FFmpegSampleRate), // 48KHz sampling rate
		"-ac", fmt.Sprintf("%d", FFmpegChannels), // Stereo channels
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
		opusEncoder, err = gopus.NewEncoder(FFmpegSampleRate, FFmpegChannels, gopus.Audio)
		if err != nil {
			log.Printf("ERROR: Failed to create Opus encoder: %v", err)
			done <- true
			return
		}

		// Set the bitrate
		opusEncoder.SetBitrate(FFmpegBitrate) // 128 kbps

		// Buffer for reading audio data
		audiobuf := make([]int16, OpusFrameSize*FFmpegChannels) // 960 samples * 2 channels

		// Send audio data to Discord
		for {
			// Check if skip was called
			if v.stop {
				log.Printf("INFO: Skip detected during existing connection playback, stopping")
				break
			}

			// Handle pause - stop sending frames but keep reading from ffmpeg
			if v.paused {
				log.Printf("DEBUG: Audio paused - not sending frames to Discord")
				// Still read from ffmpeg to prevent buffer issues, but don't send to Discord
				err = binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					log.Printf("INFO: End of audio file reached while paused")
					break
				}
				if err != nil {
					log.Printf("ERROR: Error reading from ffmpeg while paused: %v", err)
					break
				}
				// Sleep a bit to prevent busy waiting
				time.Sleep(20 * time.Millisecond)
				continue // Skip sending to Discord
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
			opus, err := opusEncoder.Encode(audiobuf, OpusFrameSize, OpusFrameSize*FFmpegChannels*2)
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
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	// Main loop - wait for audio to finish or keep alive
	for {
		select {
		case <-ticker.C:
			// Check if skip was called
			if v.stop {
				log.Printf("INFO: Skip detected during existing connection main loop, stopping")
				// Kill ffmpeg process
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
				// Set speaking to false but DON'T disconnect
				if vc != nil && vc.Ready {
					vc.Speaking(false)
				}
				return
			}

			// Handle pause/resume logic
			if v.paused {
				// Song is paused, set speaking to false and stop sending audio
				if v.voice != nil && v.voice.Ready {
					v.voice.Speaking(false)
				}
				log.Printf("DEBUG: MP3 playback paused - audio transmission will be handled in stream loop")

				// Wait for resume or skip
				for v.paused && !v.stop {
					time.Sleep(100 * time.Millisecond)
				}

				// If not stopped, resume
				if !v.stop && !v.paused {
					if v.voice != nil && v.voice.Ready {
						v.voice.Speaking(true)
					}
					log.Printf("DEBUG: MP3 playback resumed - audio transmission will resume")
				}
			} else {
				// Keep the speaking state alive if not paused
				if vc != nil && vc.Ready {
					vc.Speaking(true)
				}
			}
		case <-done:
			// Audio playback is complete, clean up
			log.Printf("INFO: Audio playback completed with existing connection")

			// Wait for ffmpeg to finish
			err = cmd.Wait()
			if err != nil {
				log.Printf("ERROR: FFMPEG exited with error: %v", err)
			}

			// Set speaking to false but DON'T disconnect - this is the key difference
			if vc != nil && vc.Ready {
				vc.Speaking(false)
			}

			return
		}
	}
}
