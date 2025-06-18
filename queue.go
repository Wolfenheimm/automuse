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
	// Prevent multiple simultaneous playQueue calls
	if v.nowPlaying != (Song{}) {
		log.Printf("WARN: playQueue called while already playing: %s", v.nowPlaying.Title)
		return
	}

	// Pre-download first 3 songs before starting playback
	queueMutex.Lock()
	initialQueue := make([]Song, len(queue))
	copy(initialQueue, queue)
	queueMutex.Unlock()

	if len(initialQueue) > 0 && !isManual {
		log.Printf("INFO: Pre-downloading initial songs before playback")
		err := bufferManager.PreDownloadInitialSongs(initialQueue, s, m.ChannelID)
		if err != nil {
			log.Printf("ERROR: Failed to pre-download songs: %v", err)
		}
	}

	// Establish voice connection once for the entire queue
	if v.voice == nil || !v.voice.Ready {
		log.Printf("INFO: Establishing voice connection for queue playback")

		// Find voice channel
		voiceChannelID, err := v.findUserVoiceChannel()
		if err != nil || voiceChannelID == "" {
			log.Printf("ERROR: Failed to find a voice channel: %v", err)
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Could not find a voice channel to join!")
			return
		}

		// Join voice channel once
		v.voice, err = s.ChannelVoiceJoin(v.guildID, voiceChannelID, false, false)
		if err != nil {
			log.Printf("ERROR: Failed to join voice channel: %v", err)
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Failed to join voice channel!")
			return
		}

		// Wait for voice connection to be ready
		ready := false
		for i := 0; i < 5; i++ {
			if v.voice != nil && v.voice.Ready {
				ready = true
				log.Printf("INFO: Voice connection ready for queue playback")
				break
			}
			log.Printf("INFO: Waiting for voice connection (attempt %d/5)", i+1)
			time.Sleep(1 * time.Second)
		}

		if !ready {
			log.Printf("ERROR: Voice connection failed to become ready")
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Voice connection failed!")
			if v.voice != nil {
				v.voice.Disconnect()
				v.voice = nil
			}
			return
		}

		log.Printf("INFO: Voice connection established successfully")
	}

	// Start the buffer manager for ongoing downloads
	bufferManager.StartBuffering(s, m.ChannelID)

	// Iterate through the queue, playing each song
	currentPlayingIndex := 0
	for {
		// Thread-safe queue access
		queueMutex.Lock()
		if len(queue) == 0 {
			queueMutex.Unlock()
			break
		}
		v.nowPlaying, queue = queue[0], queue[1:]

		// Update buffer manager with current queue state
		bufferManager.UpdateQueue(queue, currentPlayingIndex)

		// Check if there's a next song for messaging
		var hasNextSong bool
		if len(queue) > 0 {
			hasNextSong = true
		}
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
			if isManual {
				v.DCA(v.nowPlaying.Title, isManual, true)
			} else {
				v.DCA(v.nowPlaying.VideoURL, isManual, true)
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
		hasNextSong = len(queue) > 0 && queue[0].Title != ""
		var nextSongTitle string
		if hasNextSong {
			nextSongTitle = queue[0].Title
		}
		queueMutex.Unlock()

		if hasNextSong {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Next! Now playing ["+nextSongTitle+"] :loop:")
		}
	}

	// No more songs in the queue, reset and disconnect voice
	setPlaybackEnding(true) // Set flag to prevent inappropriate error messages
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Nothing left to play, peace! :v:")
	v.stop = true
	v.nowPlaying = Song{}

	queueMutex.Lock()
	queue = []Song{}
	queueMutex.Unlock()

	// Stop the buffer manager
	bufferManager.StopBuffering()

	// Disconnect voice connection only when queue is fully complete
	if v.voice != nil {
		log.Printf("INFO: Queue finished, disconnecting from voice channel")
		v.voice.Disconnect()
		v.voice = nil
	}

	// Cleanup the encoder
	if v.encoder != nil {
		v.encoder.Cleanup()
	}

	// Reset the playback ending flag after a short delay
	go func() {
		time.Sleep(2 * time.Second)
		setPlaybackEnding(false)
		setPlaybackState(false) // Reset playback state when queue finishes
	}()
}

// queueSingleSong fetches metadata and queues a single video
func queueSingleSong(m *discordgo.MessageCreate, link string) {
	log.Printf("[DEBUG] Attempting to get video from link: %s", link)

	// Extract video ID first for cache checking
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

	// Check if song is already cached
	if videoID != "" {
		if cachedMetadata, exists := metadataManager.GetSong(videoID); exists {
			log.Printf("[INFO] Found cached song: %s", cachedMetadata.Title)

			// Check for similar songs with duplicate detection
			similarSongs := metadataManager.FindSimilarSongs(cachedMetadata.Title, 0.8)
			if len(similarSongs) > 1 {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** Found %d similar songs in cache, using: [%s] :recycle:", len(similarSongs), cachedMetadata.Title))
			}

			// Create song with cached data
			song = fillSongInfo(m.ChannelID, m.Author.ID, m.ID, cachedMetadata.Title, cachedMetadata.VideoID, cachedMetadata.Duration)
			song.VideoURL = cachedMetadata.FilePath

			// Thread-safe queue append
			queueMutex.Lock()
			queue = append(queue, song)
			queueMutex.Unlock()

			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Adding cached ["+cachedMetadata.Title+"] to the Queue  :musical_note:")
			return
		}
	}

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

		youtubeErr := NewYouTubeError("Failed to get video information",
			"Failed to get video information. The video may be private, age-restricted, or unavailable in your region.", err).
			WithContext("video_url", link).
			WithContext("user_id", m.Author.ID)
		errorHandler.Handle(youtubeErr, m.ChannelID)
		return
	}

	log.Printf("[DEBUG] Successfully retrieved video: %s (ID: %s)", video.Title, video.ID)
	log.Printf("[DEBUG] Video duration: %s", video.Duration)

	// Check for similar songs in cache before downloading
	similarSongs := metadataManager.FindSimilarSongs(video.Title, 0.8)
	if len(similarSongs) > 0 {
		log.Printf("[INFO] Found %d similar songs in cache for: %s", len(similarSongs), video.Title)

		// Show user the similar songs found
		similarTitles := make([]string, len(similarSongs))
		for i, similar := range similarSongs {
			similarTitles[i] = similar.Title
		}

		if len(similarTitles) == 1 {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** :warning: Found similar song in cache: [%s]. Adding new version anyway.", similarTitles[0]))
		} else {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** :warning: Found %d similar songs in cache. Adding new version anyway.", len(similarTitles)))
		}
	}

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

		audioErr := NewAudioError("Failed to get working stream",
			"Sorry, I couldn't get a working stream for this video :(", err).
			WithContext("video_id", video.ID).
			WithContext("video_title", video.Title).
			WithContext("user_id", m.Author.ID)
		errorHandler.Handle(audioErr, m.ChannelID)
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
		selectedSong := searchQueue[input-1]
		videoURL := "https://www.youtube.com/watch?v=" + selectedSong.Id

		// Check if this song is already cached before downloading
		if cachedMetadata, exists := metadataManager.GetSong(selectedSong.Id); exists {
			log.Printf("[INFO] Found cached song from search selection: %s", cachedMetadata.Title)

			// Check for similar songs with duplicate detection
			similarSongs := metadataManager.FindSimilarSongs(cachedMetadata.Title, 0.8)
			if len(similarSongs) > 1 {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** Found %d similar songs in cache, using: [%s] :recycle:", len(similarSongs), cachedMetadata.Title))
			}

			// Create song with cached data
			song = fillSongInfo(m.ChannelID, m.Author.ID, m.ID, cachedMetadata.Title, cachedMetadata.VideoID, cachedMetadata.Duration)
			song.VideoURL = cachedMetadata.FilePath

			// Thread-safe queue append
			queueMutex.Lock()
			queue = append(queue, song)
			queueMutex.Unlock()

			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Adding cached ["+cachedMetadata.Title+"] from search to the Queue  :musical_note:")
		} else {
			// Not cached, proceed with normal download and queue
			queueSingleSong(m, videoURL)
		}
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
	// Don't show error messages if stop was recently requested or playback is ending
	if isStopRequested() || isPlaybackEnding() {
		log.Printf("[DEBUG] prepDisplayQueue skipped - stop requested: %t, playback ending: %t", isStopRequested(), isPlaybackEnding())
		return
	}

	// Debug logging to understand when this function is called inappropriately
	log.Printf("[DEBUG] prepDisplayQueue called: commData=%v, queueLenBefore=%d, currentQueueLen=%d", commData, queueLenBefore, len(queue))

	// Display queue if it grew in size (new items added)
	if queueLenBefore < len(queue) {
		log.Printf("[DEBUG] Queue grew from %d to %d, displaying updated queue", queueLenBefore, len(queue))
		displayQueue(m)
		return
	}

	// Don't show error messages for certain commands that shouldn't trigger them
	if len(commData) > 0 {
		firstCommand := strings.ToLower(commData[0])
		if firstCommand == "stop" || firstCommand == "skip" || firstCommand == "queue" {
			log.Printf("[DEBUG] Skipping error message for %s command", firstCommand)
			return
		}
	}

	// Check if this is a numeric input (search result selection)
	if len(commData) > 1 {
		if _, err := strconv.Atoi(commData[1]); err == nil {
			log.Printf("[DEBUG] Skipping error message - detected numeric input: %s", commData[1])
			return
		}
	}

	// Check if we're currently playing something - if so, we're likely not in an error state
	if v.nowPlaying != (Song{}) {
		log.Printf("[DEBUG] Skipping error message - something is currently playing")
		return
	}

	// Only show error message if we're actually trying to add content
	// Skip if this seems to be an end-of-queue or stop scenario
	// FIXED: Using thread-safe stopRequested access to prevent race conditions
	if !isStopRequested() && !isPlaybackEnding() && len(commData) > 1 && (strings.Contains(commData[1], "http") || len(commData[1]) > 3) {
		// Additional check: don't show error if the command looks like it might be successful later
		// (e.g., during playlist processing)
		if strings.Contains(commData[1], "playlist") || strings.Contains(commData[1], "list=") {
			log.Printf("[DEBUG] Skipping error message - appears to be playlist processing")
			return
		}

		log.Printf("[DEBUG] Showing 'nothing added' message for command: %v", commData)
		nothingAddedMessage := "**[Muse]** Nothing was added, playlist or song was empty...\n"
		nothingAddedMessage = nothingAddedMessage + "Note:\n"
		nothingAddedMessage = nothingAddedMessage + "- Playlists should have the following url structure: <https://www.youtube.com/playlist?list=><PLAYLIST IDENTIFIER>\n"
		nothingAddedMessage = nothingAddedMessage + "- Videos should have the following url structure: <https://www.youtube.com/watch?v=><VIDEO IDENTIFIER>\n"
		nothingAddedMessage = nothingAddedMessage + "- Youtu.be links or links set at a certain time (t=#s) have not been implemented - sorry!"
		s.ChannelMessageSend(m.ChannelID, nothingAddedMessage)
	} else {
		log.Printf("[DEBUG] Skipping error message - command doesn't seem to be adding content: %v", commData)
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

// queuePlaylistThreaded processes playlists by downloading all songs first
// then starting playback once everything is ready
func queuePlaylistThreaded(playlistID string, m *discordgo.MessageCreate) {
	// Check user-specific cooldown to prevent spam
	userKey := m.Author.ID + ":playlist"
	if isCommandActive(userKey, "playlist") {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** ⏳ Please wait a moment before adding another playlist. (User rate limit)")
		log.Printf("WARN: Playlist processing rejected - user %s rate limited", m.Author.ID)
		return
	}
	setCommandActive(userKey, "playlist")
	defer clearCommandActive(userKey, "playlist")

	// Acquire playlist processing semaphore with timeout
	select {
	case playlistSemaphore <- struct{}{}:
		// Got the semaphore, proceed
		defer func() { <-playlistSemaphore }()
	case <-time.After(5 * time.Second): // Add timeout to prevent hanging
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** 🚫 Playlist processing system is at capacity. Please try again in a moment.")
		log.Printf("WARN: Playlist processing rejected - semaphore timeout (system at capacity)")
		return
	}

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

	// Check if playlist is too large
	if len(allVideoIds) > 100 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** 🚫 Playlist too large! (%d songs). Maximum allowed is 100 songs to prevent system overload.", len(allVideoIds)))
		log.Printf("WARN: Playlist rejected - too large (%d songs)", len(allVideoIds))
		return
	}

	// Check if adding this playlist would exceed queue limit
	queueMutex.Lock()
	currentQueueSize := len(queue)
	queueMutex.Unlock()

	if currentQueueSize+len(allVideoIds) > maxQueueSize {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** 🚫 Adding this playlist (%d songs) would exceed the maximum queue size (%d). Current queue: %d songs.", len(allVideoIds), maxQueueSize, currentQueueSize))
		log.Printf("WARN: Playlist rejected - would exceed queue limit")
		return
	}

	log.Printf("INFO: Found %d videos in playlist", len(allVideoIds))
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** Found %d videos! Processing playlist (this may take a moment)... ⏳", len(allVideoIds)))

	// Process all videos and queue them before starting playback
	songsProcessed := 0
	successfullyQueued := 0

	// Reduced concurrent processing to prevent overwhelming
	maxConcurrent := 2 // Reduced from 3 to 2 for better stability
	semaphore := make(chan struct{}, maxConcurrent)

	// Structure to hold results with their original index
	type processResult struct {
		index   int
		song    Song
		success bool
	}

	// Channel to collect results from goroutines
	resultChan := make(chan processResult, len(allVideoIds))

	// Process all videos in parallel
	for i, videoId := range allVideoIds {
		// Check if we should stop processing (user might have stopped)
		if isStopRequested() || isPlaybackEnding() {
			log.Printf("INFO: Stopping playlist processing due to stop request")
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Playlist processing stopped.")
			return
		}

		semaphore <- struct{}{} // Acquire semaphore

		go func(videoId string, index int) {
			defer func() { <-semaphore }() // Release semaphore

			videoURL := "https://www.youtube.com/watch?v=" + videoId
			log.Printf("INFO: Processing video %d/%d: %s", index+1, len(allVideoIds), videoId)

			// Get video metadata first (always, even if we have cached files)
			title, videoID, duration, err := getVideoMetadata(videoId)
			if err != nil {
				log.Printf("[ERROR] Failed to get video metadata for %s: %v", videoId, err)
				resultChan <- processResult{index: index, success: false}
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

			log.Printf("[INFO] Successfully processed (%d/%d): %s", index+1, len(allVideoIds), title)
			resultChan <- processResult{index: index, song: song, success: true}

		}(videoId, i)
	}

	// Collect all results with timeout protection
	results := make([]processResult, len(allVideoIds))
	timeout := time.NewTimer(5 * time.Minute) // 5 minute timeout for playlist processing
	defer timeout.Stop()

	for i := 0; i < len(allVideoIds); i++ {
		select {
		case result := <-resultChan:
			results[result.index] = result
			songsProcessed++

			if result.success {
				successfullyQueued++
			}

			// Send progress updates less frequently to avoid spam
			if songsProcessed%10 == 0 || songsProcessed == len(allVideoIds) {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** Processed %d/%d songs from playlist... 🎵", songsProcessed, len(allVideoIds)))
			}

		case <-timeout.C:
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** ⏰ Playlist processing timed out. Some songs may not have been added.")
			log.Printf("ERROR: Playlist processing timed out after 5 minutes")
			goto processResults
		}
	}

processResults:
	log.Printf("INFO: Playlist processing complete. Successfully queued %d/%d songs", successfullyQueued, len(allVideoIds))

	if successfullyQueued == 0 {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** No songs could be processed from the playlist. All videos may be unavailable or restricted.")
		return
	}

	// Add successful songs to queue in original playlist order
	queueMutex.Lock()
	for _, result := range results {
		if result.success {
			queue = append(queue, result.song)
		}
	}
	queueMutex.Unlock()

	// Now start playback with all songs queued
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** ✅ Playlist ready! Added %d songs to queue. 🎵", successfullyQueued))

	// **ALWAYS SHOW COMPLETE QUEUE** after playlist processing
	log.Printf("INFO: Displaying complete queue after playlist processing")
	displayQueue(m)

	// **SMART PLAYBACK MANAGEMENT** - Start playback only if nothing is playing
	queueMutex.Lock()
	currentQueueSize = len(queue)
	queueMutex.Unlock()

	if v.nowPlaying == (Song{}) && currentQueueSize >= 1 && !getPlaybackState() {
		// Nothing is playing, start playback
		log.Printf("INFO: Starting playback for playlist with %d songs", currentQueueSize)
		joinVoiceChannel()
		prepFirstSongEntered(m, false)
	} else if getPlaybackState() || v.nowPlaying != (Song{}) {
		// Something is already playing, just notify that songs were added
		log.Printf("INFO: Playback already in progress, playlist songs added to queue")
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** ✅ Playlist added to queue! Songs will play after current music. 🎵")
	}

	log.Printf("INFO: Threaded playlist processing completed for %d videos", len(allVideoIds))
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
