package main

import (
	"log"
	"strconv"

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
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Nothing left to play, peace! :v:")
	v.stop = true
	v.nowPlaying = Song{}
	queue = []Song{}
	if v.encoder != nil {
		v.encoder.Cleanup()
	}
	v.voice.Disconnect()
}

// Fetch a single video and place into song queue
// Single video link (not a playlist)
func queueSingleSong(m *discordgo.MessageCreate, link string) {
	video, err := client.GetVideo(link)
	if err != nil {
		log.Println(err)
	} else {
		// Get formats with audio channels only
		format := video.Formats.WithAudioChannels()

		// Fill Song Info
		song = fillSongInfo(m.ChannelID, m.Author.ID, m.ID, video.ID, video.Title, video.Duration.String())
		var url string
		var err error

		// Select the correct video format - Check if it's in the song quality list file first
		formatList := &format[0]
		for _, value := range badQualitySongs.BadQualitySongNodes {
			if video.Title == value.Title {
				formatList = &format[value.FormatNo]
				break
			}
		}
		url, err = client.GetStreamURL(video, formatList)

		if err != nil {
			log.Println(err)
		} else {
			song.VideoURL = url
			queue = append(queue, song)
		}
	}
}

func playFromSearch(input int, m *discordgo.MessageCreate) {
	if input <= len(searchQueue) {
		queueSingleSong(m, searchQueue[input-1].Id)
		searchQueue = []SongSearch{}
	}
	searchRequested = false
}

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
		skip(m)
	} else {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Selected input was not in queue range")
	}
}

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
