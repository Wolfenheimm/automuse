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
			joinVoiceChannel(m)
			prepFirstSongEntered(m, false)
		} else if !searchRequested {
			prepDisplayQueue(commData, queueLenBefore, m)
		}

		commDataIsValid = false
	}
}

func queueKudasai(m *discordgo.MessageCreate) {
	commData := []string{"queue", "https://www.youtube.com/watch?v=35AgDDPQE48"}
	prepWatchCommand(commData, m)
}

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
		joinVoiceChannel(m)
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
