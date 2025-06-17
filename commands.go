package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Get & queue audio in a YouTube video / playlist
func queueSong(m *discordgo.MessageCreate) {
	// Prevent queue processing if stop was recently requested
	if isStopRequested() {
		return
	}

	commData, commDataIsValid := sanitizeQueueSongInputs(m)
	queueLenBefore := len(queue)
	playbackAlreadyStarted := false

	if !commDataIsValid {
		err := NewValidationError("Invalid command format", nil).
			WithContext("command", m.Content).
			WithContext("user_id", m.Author.ID)
		errorHandler.Handle(err, m.ChannelID)
		return
	}

	// Clear stop flag when starting new queue operation
	setStopRequested(false)

	// Check if a youtube link is present
	if strings.Contains(m.Content, "https://www.youtube") {
		// Check if the link is a playlist or a simple video
		if strings.Contains(m.Content, "list") && strings.Contains(m.Content, "-pl") || strings.Contains(m.Content, "/playlist?") {
			prepPlaylistCommand(commData, m)
		} else if strings.Contains(m.Content, "watch") && !strings.Contains(m.Content, "-pl") {
			playbackAlreadyStarted = prepWatchCommand(commData, m)
		}
		resetSearch() // In case a search was called prior to this
	} else {
		// Search or queue input was sent
		prepSearchQueueSelector(commData, m)
	}

	// If there's nothing playing and the queue grew AND playback wasn't already started
	if !playbackAlreadyStarted && v.nowPlaying == (Song{}) && len(queue) >= 1 {
		if err := joinVoiceChannelWithError(); err != nil {
			voiceErr := NewVoiceError("Failed to join voice channel",
				"Could not join voice channel. Please check permissions.", err).
				WithContext("guild_id", v.guildID).
				WithContext("user_id", m.Author.ID)
			errorHandler.Handle(voiceErr, m.ChannelID)
			return
		}
		prepFirstSongEntered(m, false)
	} else if !searchRequested && !isStopRequested() {
		prepDisplayQueue(commData, queueLenBefore, m)
	}
}

// Helper function for voice channel joining with error handling
func joinVoiceChannelWithError() error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in joinVoiceChannel: %v", r)
		}
	}()

	joinVoiceChannel()

	// Check if voice connection was successful
	if v.voice == nil {
		return NewVoiceError("Voice connection is nil after join attempt", "", nil)
	}

	return nil
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
	setStopRequested(true) // Set flag to prevent additional queue processing
	resetSearch()

	// Stop buffer manager
	bufferManager.StopBuffering()

	if v.encoder != nil {
		v.encoder.Cleanup()
	}

	if v.voice != nil {
		v.voice.Disconnect()
	}

	// Reset the stop flag after everything has stopped
	setStopRequested(false)
}

// Skips the current song
func skip(m *discordgo.MessageCreate) {
	// Check if skipping current song or skipping to another song
	if m.Content == "skip" {
		if v.nowPlaying == (Song{}) {
			err := NewQueueError("No song currently playing", "Queue is empty - There's nothing to skip!", nil).
				WithContext("user_id", m.Author.ID)
			errorHandler.Handle(err, m.ChannelID)
			return
		}

		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Skipping "+v.nowPlaying.Title+" :loop:")

		// Show what's playing next
		queueMutex.Lock()
		if len(queue) > 0 {
			nextSong := queue[0]
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Next! Now playing ["+nextSong.Title+"] :notes:")
		} else {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** No more songs in queue after this skip.")
		}
		queueMutex.Unlock()

		prepSkip()
		resetSearch()
		log.Println("Skipped " + v.nowPlaying.Title)
	} else if strings.Contains(m.Content, "skip to ") || (strings.HasPrefix(m.Content, "skip ") && m.Content != "skip") {
		msgData := strings.Split(m.Content, " ")
		var targetPosition int
		var err error

		// Handle both \"skip to X\" and \"skip X\" formats
		if strings.Contains(m.Content, "skip to ") {
			// Can only accept 3 params: skip to #
			if len(msgData) != 3 {
				validationErr := NewValidationError("Invalid format. Use 'skip to [number]' or 'skip [number]'", nil).
					WithContext("command", m.Content).
					WithContext("user_id", m.Author.ID)
				errorHandler.Handle(validationErr, m.ChannelID)
				return
			}
			targetPosition, err = strconv.Atoi(msgData[2])
		} else {
			// Handle \"skip X\" format
			if len(msgData) != 2 {
				validationErr := NewValidationError("Invalid format. Use 'skip [number]' or 'skip to [number]'", nil).
					WithContext("command", m.Content).
					WithContext("user_id", m.Author.ID)
				errorHandler.Handle(validationErr, m.ChannelID)
				return
			}
			targetPosition, err = strconv.Atoi(msgData[1])
		}

		// Validate the number
		if err != nil {
			validationErr := NewValidationError("Please provide a valid number for the skip position.", err).
				WithContext("command", m.Content).
				WithContext("user_id", m.Author.ID)
			errorHandler.Handle(validationErr, m.ChannelID)
			return
		}

		// Check if target position exists in queue
		queueMutex.Lock()
		queueLength := len(queue)
		queueMutex.Unlock()

		if targetPosition <= 0 || targetPosition > queueLength {
			queueErr := NewQueueError("Invalid queue position",
				fmt.Sprintf("Position %d doesn't exist in the queue. Queue has %d songs.", targetPosition, queueLength), nil).
				WithContext("target_position", targetPosition).
				WithContext("queue_length", queueLength).
				WithContext("user_id", m.Author.ID)
			errorHandler.Handle(queueErr, m.ChannelID)
			return
		}

		// Skip to the target position
		queueMutex.Lock()
		var tmp []Song
		for i, value := range queue {
			if i >= targetPosition-1 {
				tmp = append(tmp, value)
			}
		}
		targetSong := queue[targetPosition-1]
		queue = tmp
		queueMutex.Unlock()

		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Jumping to ["+targetSong.Title+"] (position "+strconv.Itoa(targetPosition)+") :leftwards_arrow_with_hook:")
		log.Printf("Jumping to [%s] at position %d", targetSong.Title, targetPosition)

		prepSkip()
		resetSearch()
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

// Shows help menu with all available commands
func showHelp(m *discordgo.MessageCreate) {
	helpMessage := ":robot: **[Muse] HELP MENU** :robot:\n\n"
	helpMessage += ":musical_note: **MUSIC COMMANDS** :musical_note:\n"
	helpMessage += "`play [YouTube URL]` - Play a YouTube video or playlist\n"
	helpMessage += "`play [search term]` - Search for and play a song\n"
	helpMessage += "`play stuff` - Queue all local MP3 files from mpegs folder\n"
	helpMessage += "`stop` - Stop current song and clear the queue\n"
	helpMessage += "`skip` - Skip the current song\n"
	helpMessage += "`skip [number]` - Skip to a specific position in queue\n"
	helpMessage += "`skip to [number]` - Skip to a specific position in queue\n"
	helpMessage += "`queue` - Display the current queue\n"
	helpMessage += "`remove [number]` - Remove a song from the queue at position\n"
	helpMessage += "`move [from] [to]` - Move a song from one position to another\n"
	helpMessage += "`shuffle` - Shuffle the current queue\n"
	helpMessage += "`cache` - Show cache statistics and information\n"
	helpMessage += "`cache-clear` - Clear old cached songs (older than 7 days)\n"
	helpMessage += "`buffer-status` - Show buffer manager status and download queue\n\n"
	helpMessage += ":gear: **SUPPORTED FORMATS** :gear:\n"
	helpMessage += "• YouTube videos: `https://www.youtube.com/watch?v=...`\n"
	helpMessage += "• YouTube playlists: `https://www.youtube.com/playlist?list=...`\n"
	helpMessage += "• Search terms: `play [artist] - [song title]`\n\n"
	helpMessage += ":information_source: **EXAMPLES** :information_source:\n"
	helpMessage += "`play https://www.youtube.com/watch?v=dQw4w9WgXcQ`\n"
	helpMessage += "`play never gonna give you up`\n"
	helpMessage += "`skip 3` - Skip to song #3 in queue\n"
	helpMessage += "`remove 2` - Remove song #2 from queue\n\n"

	s.ChannelMessageSend(m.ChannelID, helpMessage)
}

func cacheStatsCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	stats := metadataManager.GetDetailedStats()

	// Calculate cache size on disk
	var totalCacheSize int64
	if files, err := filepath.Glob("downloads/*.mp3"); err == nil {
		for _, file := range files {
			if info, err := os.Stat(file); err == nil {
				totalCacheSize += info.Size()
			}
		}
	}

	// Format cache size
	cacheSizeStr := formatBytes(totalCacheSize)

	response := "**[Muse]** :floppy_disk: **Cache Statistics** :floppy_disk:\n\n"
	response += fmt.Sprintf(":musical_note: **Total Songs Cached:** %d\n", stats.TotalSongs)
	response += fmt.Sprintf(":minidisc: **Total Cache Size:** %s\n", cacheSizeStr)
	response += fmt.Sprintf(":chart_with_upwards_trend: **Total Plays:** %d\n", stats.TotalPlays)
	response += fmt.Sprintf(":arrow_forward: **Average Plays per Song:** %.1f\n\n", stats.AverageUsage)

	if len(stats.TopSongs) > 0 {
		response += ":crown: **Most Played Songs:**\n"
		for i, song := range stats.TopSongs {
			if i >= 5 {
				break
			} // Show top 5
			response += fmt.Sprintf("%d. [%s] - %d plays\n", i+1, song.Title, song.UseCount)
		}
	}

	s.ChannelMessageSend(m.ChannelID, response)
}

func cacheClearCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Get songs older than 7 days
	oldSongs := metadataManager.GetOldSongs(7 * 24 * time.Hour)

	if len(oldSongs) == 0 {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** :wastebasket: No old cache files to clean up!")
		return
	}

	var totalSize int64
	removedCount := 0

	for _, song := range oldSongs {
		// Check if file exists and get size
		if info, err := os.Stat(song.FilePath); err == nil {
			totalSize += info.Size()
			// Remove the file
			if err := os.Remove(song.FilePath); err == nil {
				// Remove from metadata
				metadataManager.RemoveSong(song.VideoID)
				removedCount++
			}
		}
	}

	// Save metadata after cleanup
	metadataManager.SaveMetadata()

	response := "**[Muse]** :wastebasket: **Cache Cleanup Complete!**\n"
	response += fmt.Sprintf(":file_folder: **Files Removed:** %d\n", removedCount)
	response += fmt.Sprintf(":minidisc: **Space Freed:** %s\n", formatBytes(totalSize))

	s.ChannelMessageSend(m.ChannelID, response)
}

func bufferStatusCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	bufferManager.mutex.RLock()
	defer bufferManager.mutex.RUnlock()

	response := "**[Muse]** :gear: **Buffer Manager Status** :gear:\n\n"
	response += fmt.Sprintf(":green_circle: **Active:** %t\n", bufferManager.isActive)
	response += fmt.Sprintf(":musical_note: **Buffer Size:** %d songs\n", bufferManager.maxBuffer)
	response += fmt.Sprintf(":arrow_down: **Download Queue:** %d songs\n", len(bufferManager.downloadQueue))
	response += fmt.Sprintf(":hourglass: **Currently Downloading:** %d songs\n\n", len(bufferManager.downloading))

	if len(bufferManager.downloadQueue) > 0 {
		response += ":clock1: **Next Songs in Buffer:**\n"
		for i, song := range bufferManager.downloadQueue {
			if i >= 3 {
				break
			} // Show next 3
			status := ":white_circle:"
			if bufferManager.downloading[song.VidID] {
				status = ":orange_circle: Downloading..."
			} else if metadataManager.HasSong(song.VidID) {
				status = ":green_circle: Cached"
			}
			response += fmt.Sprintf("%d. [%s] %s\n", i+1, song.Title, status)
		}
	}

	s.ChannelMessageSend(m.ChannelID, response)
}

// Helper function to format bytes into human-readable sizes
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func moveQueueCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Usage: `move [from] [to]` - Move song from position to position")
		return
	}

	fromPos, err1 := strconv.Atoi(args[0])
	toPos, err2 := strconv.Atoi(args[1])

	if err1 != nil || err2 != nil {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Please provide valid position numbers")
		return
	}

	queueMutex.Lock()
	defer queueMutex.Unlock()

	if fromPos < 1 || fromPos > len(queue) || toPos < 1 || toPos > len(queue) {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** Position must be between 1 and %d", len(queue)))
		return
	}

	// Convert to 0-based indexing
	fromPos--
	toPos--

	// Move the song
	song := queue[fromPos]
	// Remove from original position
	queue = append(queue[:fromPos], queue[fromPos+1:]...)
	// Insert at new position
	if toPos > len(queue) {
		toPos = len(queue)
	}
	queue = append(queue[:toPos], append([]Song{song}, queue[toPos:]...)...)

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** :arrow_right: Moved [%s] to position %d", song.Title, toPos+1))
}

func shuffleQueueCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	queueMutex.Lock()
	defer queueMutex.Unlock()

	if len(queue) <= 1 {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** :twisted_rightwards_arrows: Queue needs at least 2 songs to shuffle")
		return
	}

	// Simple shuffle algorithm
	for i := len(queue) - 1; i > 0; i-- {
		j := i % (i + 1) // Simple pseudo-random
		if j != i {
			queue[i], queue[j] = queue[j], queue[i]
		}
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** :twisted_rightwards_arrows: Shuffled %d songs in the queue!", len(queue)))
}
