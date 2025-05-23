package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	yt "github.com/kkdai/youtube/v2"
)

// This is the main function that plays the queue
// - It will play the queue until it's empty
// - If the queue is empty, it will leave the voice channel
func playQueue(m *discordgo.MessageCreate, isManual bool) {
	// Iterate through the queue, playing each song
	for {
		// Thread-safe queue access
		queueMutex.Lock()
		if len(queue) == 0 {
			queueMutex.Unlock()
			break
		}
		v.nowPlaying, queue = queue[0], queue[1:]
		queueMutex.Unlock()

		log.Printf("INFO: Starting playback of: %s", v.nowPlaying.Title)

		// Reset stop flag for this song
		v.stop = false

		if v.voice != nil {
			v.voice.Speaking(true)
		}

		// Create a channel to signal when audio playback is complete
		audioComplete := make(chan bool, 1)

		// Start audio playback in a separate goroutine
		go func() {
			// TODO: Consider removing mpeg support
			if isManual {
				v.DCA(v.nowPlaying.Title, isManual)
			} else {
				v.DCA(v.nowPlaying.VideoURL, isManual)
			}
			audioComplete <- true
		}()

		// Monitor for skip commands while audio plays
		skipDetected := false
		ticker := time.NewTicker(100 * time.Millisecond)

	monitorLoop:
		for {
			select {
			case <-ticker.C:
				// Check if skip was called
				if v.stop {
					log.Printf("INFO: Skip detected in playQueue monitor, stopping current song")
					skipDetected = true
					ticker.Stop()
					break monitorLoop
				}
			case <-audioComplete:
				// Audio finished normally
				log.Printf("INFO: Audio playback completed normally")
				ticker.Stop()
				break monitorLoop
			}
		}

		if skipDetected {
			log.Printf("INFO: Skip detected, moving to next song")
			// Give a moment for cleanup
			time.Sleep(100 * time.Millisecond)
			continue // Skip to next song
		}

		// Song completed normally, show next song message if queue not empty
		queueMutex.Lock()
		hasNextSong := len(queue) > 0 && queue[0].Title != ""
		var nextSongTitle string
		if hasNextSong {
			nextSongTitle = queue[0].Title
		}
		queueMutex.Unlock()

		if hasNextSong {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Next! Now playing ["+nextSongTitle+"] :loop:")
		}
	}

	// No more songs in the queue, reset the queue + leave channel
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Nothing left to play, peace! :v:")
	v.stop = true
	v.nowPlaying = Song{}

	queueMutex.Lock()
	queue = []Song{}
	queueMutex.Unlock()

	if v.voice != nil {
		v.voice.Disconnect()
	}

	// Cleanup the encoder
	if v.encoder != nil {
		v.encoder.Cleanup()
	}
}

// queueSingleSong fetches metadata and queues a single video
func queueSingleSong(m *discordgo.MessageCreate, link string) {
	log.Printf("[DEBUG] Attempting to get video from link: %s", link)

	// First try to get video metadata using YouTube client
	video, err := client.GetVideo(link)
	if err != nil {
		log.Printf("[ERROR] Failed to get video with YouTube client: %v", err)

		// Check if it's an age restriction or embedding disabled error
		if strings.Contains(err.Error(), "age restriction") || strings.Contains(err.Error(), "embedding") || strings.Contains(err.Error(), "disabled") {
			log.Printf("[INFO] Attempting fallback with yt-dlp for restricted video")
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Video has restrictions, trying alternative method...")

			// Try using yt-dlp as fallback for restricted videos
			if queueWithYtDlp(m, link) {
				return // Success with yt-dlp
			}
		}

		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Failed to get video information. The video may be private, age-restricted, or unavailable in your region.")
		return
	}

	log.Printf("[DEBUG] Successfully retrieved video: %s (ID: %s)", video.Title, video.ID)
	log.Printf("[DEBUG] Video duration: %s", video.Duration)

	// Always create song with proper metadata first
	song = fillSongInfo(m.ChannelID, m.Author.ID, m.ID, video.Title, video.ID, video.Duration.String())

	// Now try to get the stream URL or use cached file
	url, err := getStreamURL(video.ID)
	if err != nil {
		log.Printf("[ERROR] Failed to get stream URL: %v", err)

		// Try yt-dlp fallback if stream URL fails
		log.Printf("[INFO] Stream URL failed, trying yt-dlp fallback")
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Stream access failed, trying alternative download method...")
		if queueWithYtDlp(m, link) {
			return // Success with yt-dlp
		}

		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Sorry, I couldn't get a working stream for this video :(")
		return
	}

	// Set the video URL (could be stream URL or link to original video for yt-dlp processing)
	song.VideoURL = url

	// Thread-safe queue append
	queueMutex.Lock()
	queue = append(queue, song)
	queueMutex.Unlock()

	// Message the user
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Adding ["+video.Title+"] to the Queue  :musical_note:")
}

// Queue the playlist - Gets the playlist ID and searches for all individual videos & queue's them
func queuePlaylist(playlistID string, m *discordgo.MessageCreate) {
	nextPageToken := "" // Used to iterate through videos in a playlist

	for {
		// Retrieve next set of items in the playlist.
		var snippet = []string{"snippet"}
		playlistResponse := playlistItemsList(service, snippet, playlistID, nextPageToken)

		for _, playlistItem := range playlistResponse.Items {
			videoId := playlistItem.Snippet.ResourceId.VideoId
			content := "https://www.youtube.com/watch?v=" + videoId

			// Get Video Data
			video, err := client.GetVideo(content)
			if err != nil {
				log.Println(err)
			} else {
				format := video.Formats.WithAudioChannels() // Get matches with audio channels only
				song = fillSongInfo(m.ChannelID, m.Author.ID, m.ID, video.Title, video.ID, video.Duration.String())
				formatList := prepSongFormat(format)
				url, err := client.GetStreamURL(video, formatList)

				if err != nil {
					log.Println(err)
				} else {
					song.VideoURL = url
					queue = append(queue, song)
				}
			}
		}

		// Set the token to retrieve the next page of results
		nextPageToken = playlistResponse.NextPageToken

		// Nothing left, break out
		if nextPageToken == "" {
			break
		}
	}
}

// Plays the chosen song from a list provided by the search function
func playFromSearch(input int, m *discordgo.MessageCreate) {
	if input <= len(searchQueue) && input > 0 {
		queueSingleSong(m, searchQueue[input-1].Id)
	} else {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** The value you entered was outside the range of the search...")
	}
	searchRequested = false
}

// Plays the chosen song from the queue
func playFromQueue(input int, m *discordgo.MessageCreate) {
	if input <= len(queue) && input > 0 {
		var tmp []Song
		for i, value := range queue {
			switch i {
			case 0:
				tmp = append(tmp, queue[input-1])
				tmp = append(tmp, value)
			case input - 1:
			default:
				tmp = append(tmp, value)
			}
		}
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Moved "+queue[input-1].Title+" to the top of the queue")
		queue = tmp
		prepSkip()
	} else {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Selected input was not in queue range")
	}
}

// Prepares queue display
func prepDisplayQueue(commData []string, queueLenBefore int, m *discordgo.MessageCreate) {
	// Only display queue if it grew in size...
	if queueLenBefore < len(queue) {
		displayQueue(m)
	} else {
		if _, err := strconv.Atoi(commData[1]); err == nil {
			return
		}

		nothingAddedMessage := "**[Muse]** Nothing was added, playlist or song was empty...\n"
		nothingAddedMessage = nothingAddedMessage + "Note:\n"
		nothingAddedMessage = nothingAddedMessage + "- Playlists should have the following url structure: <https://www.youtube.com/playlist?list=><PLAYLIST IDENTIFIER>\n"
		nothingAddedMessage = nothingAddedMessage + "- Videos should have the following url structure: <https://www.youtube.com/watch?v=><VIDEO IDENTIFIER>\n"
		nothingAddedMessage = nothingAddedMessage + "- Youtu.be links or links set at a certain time (t=#s) have not been implemented - sorry!"
		s.ChannelMessageSend(m.ChannelID, nothingAddedMessage)
	}
}

// queueWithYtDlp uses yt-dlp as a fallback for restricted videos
func queueWithYtDlp(m *discordgo.MessageCreate, link string) bool {
	// Extract video ID from the link
	var videoID string
	if strings.Contains(link, "youtube.com/watch?v=") {
		parts := strings.Split(link, "v=")
		if len(parts) > 1 {
			videoID = strings.Split(parts[1], "&")[0]
		}
	} else if strings.Contains(link, "youtu.be/") {
		parts := strings.Split(link, "youtu.be/")
		if len(parts) > 1 {
			videoID = strings.Split(parts[1], "?")[0]
		}
	}

	if videoID == "" {
		log.Printf("[ERROR] Could not extract video ID from URL: %s", link)
		return false
	}

	log.Printf("[INFO] Using yt-dlp fallback for video ID: %s", videoID)

	// Use yt-dlp to get video info
	cmd := exec.Command("yt-dlp", "--no-download", "--print", "title", "--print", "duration", link)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[ERROR] yt-dlp failed to get video info: %v", err)
		return false
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		log.Printf("[ERROR] yt-dlp returned unexpected output format")
		return false
	}

	title := strings.TrimSpace(lines[0])
	duration := strings.TrimSpace(lines[1])

	log.Printf("[INFO] yt-dlp got video info: %s (duration: %s)", title, duration)

	// For yt-dlp fallback, we'll use the download approach since streaming might not work
	// Create the song entry with a special flag to indicate it needs yt-dlp download
	song = fillSongInfo(m.ChannelID, m.Author.ID, m.ID, title, videoID, duration)
	song.VideoURL = link // Store original URL for yt-dlp processing

	// Thread-safe queue append
	queueMutex.Lock()
	queue = append(queue, song)
	queueMutex.Unlock()

	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Adding ["+title+"] to the Queue (using fallback method) :musical_note:")
	log.Printf("[INFO] Successfully queued restricted video using yt-dlp: %s", title)

	return true
}

// queuePlaylistThreaded processes playlists by starting the first song immediately
// and downloading the rest in background threads
func queuePlaylistThreaded(playlistID string, m *discordgo.MessageCreate) {
	log.Printf("INFO: Starting threaded playlist processing for: %s", playlistID)

	var allVideoIds []string
	nextPageToken := ""

	// First pass: collect all video IDs from the playlist
	for {
		var snippet = []string{"snippet"}
		playlistResponse := playlistItemsList(service, snippet, playlistID, nextPageToken)

		for _, playlistItem := range playlistResponse.Items {
			videoId := playlistItem.Snippet.ResourceId.VideoId
			allVideoIds = append(allVideoIds, videoId)
		}

		nextPageToken = playlistResponse.NextPageToken
		if nextPageToken == "" {
			break
		}
	}

	if len(allVideoIds) == 0 {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** No videos found in playlist or playlist is private.")
		return
	}

	log.Printf("INFO: Found %d videos in playlist", len(allVideoIds))
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** Found %d videos! Starting first song, processing rest in background...", len(allVideoIds)))

	// Process the first video immediately to start playback
	firstVideoURL := "https://www.youtube.com/watch?v=" + allVideoIds[0]
	log.Printf("INFO: Processing first video immediately: %s", allVideoIds[0])

	// Queue the first song using the regular method
	queueSingleSong(m, firstVideoURL)

	// Note: Playback will be started by the main queueSong function, not here

	// Process remaining videos in background if there are more than 1
	if len(allVideoIds) > 1 {
		remainingVideos := allVideoIds[1:]
		log.Printf("INFO: Starting background processing for %d remaining videos", len(remainingVideos))

		go func() {
			songsProcessed := 1 // First song already processed

			// Use a semaphore to limit concurrent processing
			maxConcurrent := 3
			semaphore := make(chan struct{}, maxConcurrent)

			// Process remaining videos in parallel
			for i, videoId := range remainingVideos {
				semaphore <- struct{}{} // Acquire semaphore

				go func(videoId string, index int) {
					defer func() { <-semaphore }() // Release semaphore

					videoURL := "https://www.youtube.com/watch?v=" + videoId
					log.Printf("INFO: Background processing video %d/%d: %s", index+2, len(allVideoIds), videoId)

					// Get video metadata first (always, even if we have cached files)
					title, videoID, duration, err := getVideoMetadata(videoId)
					if err != nil {
						log.Printf("[ERROR] Failed to get video metadata for %s: %v", videoId, err)
						return
					}

					// Create song with proper metadata
					song := fillSongInfo(m.ChannelID, m.Author.ID, m.ID, title, videoID, duration)

					// Try to get stream URL
					url, err := getStreamURL(videoID)
					if err != nil {
						log.Printf("[ERROR] Failed to get stream URL for %s: %v", title, err)
						// Store original URL for yt-dlp processing as fallback
						song.VideoURL = videoURL
					} else {
						song.VideoURL = url
					}

					// Thread-safe queue append
					queueMutex.Lock()
					queue = append(queue, song)
					queueMutex.Unlock()

					songsProcessed++

					log.Printf("[INFO] Background queued (%d/%d): %s", songsProcessed, len(allVideoIds), title)

					// Send update every 5 songs or on completion
					if songsProcessed%5 == 0 || songsProcessed == len(allVideoIds) {
						s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** Processed %d/%d songs from playlist :musical_note:", songsProcessed, len(allVideoIds)))
					}

				}(videoId, i)
			}
		}()
	}

	log.Printf("INFO: Threaded playlist processing initiated for %d videos", len(allVideoIds))
}

// queueWithYtDlpBackground is a background-safe version that doesn't trigger playback
func queueWithYtDlpBackground(link, channelID, authorID, messageID string) bool {
	// Extract video ID from the link
	var videoID string
	if strings.Contains(link, "youtube.com/watch?v=") {
		parts := strings.Split(link, "v=")
		if len(parts) > 1 {
			videoID = strings.Split(parts[1], "&")[0]
		}
	} else if strings.Contains(link, "youtu.be/") {
		parts := strings.Split(link, "youtu.be/")
		if len(parts) > 1 {
			videoID = strings.Split(parts[1], "?")[0]
		}
	}

	if videoID == "" {
		log.Printf("[ERROR] Could not extract video ID from URL: %s", link)
		return false
	}

	log.Printf("[INFO] Using yt-dlp fallback for video ID: %s", videoID)

	// Use yt-dlp to get video info
	cmd := exec.Command("yt-dlp", "--no-download", "--print", "title", "--print", "duration", link)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[ERROR] yt-dlp failed to get video info: %v", err)
		return false
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		log.Printf("[ERROR] yt-dlp returned unexpected output format")
		return false
	}

	title := strings.TrimSpace(lines[0])
	duration := strings.TrimSpace(lines[1])

	log.Printf("[INFO] yt-dlp got video info: %s (duration: %s)", title, duration)

	// Create the song entry
	song := fillSongInfo(channelID, authorID, messageID, title, videoID, duration)
	song.VideoURL = link // Store original URL for yt-dlp processing

	// Thread-safe queue append
	queueMutex.Lock()
	queue = append(queue, song)
	queueMutex.Unlock()

	log.Printf("[INFO] Successfully queued restricted video using yt-dlp: %s", title)
	return true
}

// getVideoMetadata fetches video metadata (title, duration) for a given video ID or URL
func getVideoMetadata(videoIDOrURL string) (title, videoID, duration string, err error) {
	var video *yt.Video

	// Try YouTube client first
	if strings.HasPrefix(videoIDOrURL, "http") {
		video, err = client.GetVideo(videoIDOrURL)
	} else {
		video, err = client.GetVideo("https://www.youtube.com/watch?v=" + videoIDOrURL)
	}

	if err == nil {
		return video.Title, video.ID, video.Duration.String(), nil
	}

	// If YouTube client fails, try yt-dlp for metadata only
	log.Printf("[DEBUG] YouTube client failed for %s, trying yt-dlp: %v", videoIDOrURL, err)

	var url string
	if strings.HasPrefix(videoIDOrURL, "http") {
		url = videoIDOrURL
		// Extract video ID from URL
		if strings.Contains(url, "youtube.com/watch?v=") {
			parts := strings.Split(url, "v=")
			if len(parts) > 1 {
				videoID = strings.Split(parts[1], "&")[0]
			}
		} else if strings.Contains(url, "youtu.be/") {
			parts := strings.Split(url, "youtu.be/")
			if len(parts) > 1 {
				videoID = strings.Split(parts[1], "?")[0]
			}
		}
	} else {
		videoID = videoIDOrURL
		url = "https://www.youtube.com/watch?v=" + videoID
	}

	// Use yt-dlp to get metadata
	cmd := exec.Command("yt-dlp", "--no-download", "--print", "title", "--print", "duration", url)
	output, cmdErr := cmd.Output()
	if cmdErr != nil {
		return "", videoID, "", fmt.Errorf("both YouTube client and yt-dlp failed: %v, %v", err, cmdErr)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return "", videoID, "", fmt.Errorf("yt-dlp returned unexpected output format")
	}

	title = strings.TrimSpace(lines[0])
	duration = strings.TrimSpace(lines[1])

	return title, videoID, duration, nil
}
