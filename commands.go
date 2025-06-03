package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Get & queue audio in a YouTube video / playlist
func queueSong(m *discordgo.MessageCreate) {
	commData, commDataIsValid := sanitizeQueueSongInputs(m)
	queueLenBefore := len(queue)
	playbackAlreadyStarted := false

	if commDataIsValid {
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
		}
	} else if strings.Contains(m.Content, "skip to ") || (strings.HasPrefix(m.Content, "skip ") && m.Content != "skip") {
		msgData := strings.Split(m.Content, " ")
		var targetPosition int
		var err error

		// Handle both \"skip to X\" and \"skip X\" formats
		if strings.Contains(m.Content, "skip to ") {
			// Can only accept 3 params: skip to #
			if len(msgData) == 3 {
				targetPosition, err = strconv.Atoi(msgData[2])
			} else {
				s.ChannelMessageSend(m.ChannelID, "**[Muse]** Invalid format. Use 'skip to [number]' or 'skip [number]'")
				return
			}
		} else {
			// Handle \"skip X\" format
			if len(msgData) == 2 {
				targetPosition, err = strconv.Atoi(msgData[1])
			} else {
				s.ChannelMessageSend(m.ChannelID, "**[Muse]** Invalid format. Use 'skip [number]' or 'skip to [number]'")
				return
			}
		}

		// Validate the number
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Please provide a valid number for the skip position.")
			return
		}

		// Check if target position exists in queue
		queueMutex.Lock()
		queueLength := len(queue)
		queueMutex.Unlock()

		if targetPosition <= 0 || targetPosition > queueLength {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**[Muse]** Position %d doesn't exist in the queue. Queue has %d songs.", targetPosition, queueLength))
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
	helpMessage += "`remove [number]` - Remove a song from the queue at position\n\n"
	helpMessage += ":gear: **SUPPORTED FORMATS** :gear:\n"
	helpMessage += "• YouTube videos: `https://www.youtube.com/watch?v=...`\n"
	helpMessage += "• YouTube playlists: `https://www.youtube.com/playlist?list=...`\n"
	helpMessage += "• Search terms: `play [artist] - [song title]`\n\n"
	helpMessage += ":information_source: **EXAMPLES** :information_source:\n"
	helpMessage += "`play https://www.youtube.com/watch?v=dQw4w9WgXcQ`\n"
	helpMessage += "`play never gonna give you up`\n"
	helpMessage += "`skip 3` - Skip to song #3 in queue\n"
	helpMessage += "`remove 2` - Remove song #2 from queue\n\n"
	helpMessage += ":wave: **Need help?** Use `play help` to see this menu again!"

	s.ChannelMessageSend(m.ChannelID, helpMessage)
}
