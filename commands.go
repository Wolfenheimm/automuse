package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
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
			prepFirstSongEntered(m)
		} else if !searchRequested {
			prepDisplayQueue(commData, queueLenBefore, m)
		}
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
	// Check if a song is playing - If no song, skip this and notify
	var replyMessage string
	if v.nowPlaying == (Song{}) {
		replyMessage = "**[Muse]** Queue is empty - There's nothing to skip!"
	} else {
		replyMessage = fmt.Sprintf("**[Muse]** Skipping [%s] :loop:", v.nowPlaying.Title)
		v.stop = true
		v.speaking = false
		v.encoder.Cleanup()
		log.Println("Skipping " + v.nowPlaying.Title)
		log.Println("Queue Length: ", len(queue)-1)
	}

	resetSearch()
	s.ChannelMessageSend(m.ChannelID, replyMessage)
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
