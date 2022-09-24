package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Plays the queue
func playQueue(m *discordgo.MessageCreate) {
	for len(queue) > 0 {
		if len(queue) != 0 {
			v.nowPlaying, queue = queue[0], queue[1:]
		} else {
			v.nowPlaying = Song{}
			return
		}

		v.stop = false
		v.voice.Speaking(true)
		v.DCA(v.nowPlaying.VideoURL)

		// Queue not empty, next song isn't empty (incase nil song in queue)
		if len(queue) != 0 && queue[0].Title != "" {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Next! Now playing ["+queue[0].Title+"] :loop:")
		}
	}

	// No more songs in the queue, reset the queue + leave channel
	v.stop = true
	v.nowPlaying = Song{}
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Nothing left to play, peace! :v:")
	v.voice.Disconnect()
}

// Fetch a single video and place into song queue
// Single video link (not a playlist)
func getAndQueueSingleSong(m *discordgo.MessageCreate, link string) {
	video, err := client.GetVideo(link)
	if err != nil {
		log.Println(err)
	} else {
		// Get formats with audio channels only
		format := video.Formats.WithAudioChannels()

		// Fill Song Info
		song = fillSongInfo(m.ChannelID, m.Author.ID, m.ID, video.ID, video.Title, video.Duration.String())

		url, err := client.GetStreamURL(video, &format[0])
		if err != nil {
			log.Println(err)
		} else {
			song.VideoURL = url
			queue = append(queue, song)
		}
	}
}

// Fetches and displays the queue
func getQueue(m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Fetching Queue...")
	queueList := ":musical_note:   QUEUE LIST   :musical_note:\n"

	if v.nowPlaying != (Song{}) {
		queueList = queueList + "Now Playing: " + v.nowPlaying.Title + "  ->  Queued by <@" + v.nowPlaying.User + "> \n"
	}

	for index, element := range queue {
		queueList = queueList + " " + strconv.Itoa(index+1) + ". " + element.Title + "  ->  Queued by <@" + element.User + "> \n"
		if index+1 == 14 {
			s.ChannelMessageSend(m.ChannelID, queueList)
			queueList = ""
		}
	}

	s.ChannelMessageSend(m.ChannelID, queueList)
	log.Println(queueList)
}

// Removes a song from the queue at a specific position
func removeFromQueue(m *discordgo.MessageCreate) {
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
