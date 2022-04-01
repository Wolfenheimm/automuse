package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Get & queue all videos in a YouTube Playlist
func queueSong(message string, m *discordgo.MessageCreate, v *VoiceInstance, channelID string, alreadyInChannel bool) {

	// Split the message to get YT link
	commData := strings.Split(message, " ")

	if len(commData) == 2 {
		// If playlist.... TODO: Error checking on the link
		if strings.Contains(m.Content, "list") {
			playlistID := strings.Replace(commData[1], "https://www.youtube.com/playlist?list=", "", -1)
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Queueing Your PlayList... :infinity:")
			go queuePlaylist(playlistID, m)
		} else {
			// Single video link (not a playlist)
			video, err := client.GetVideo(commData[1])
			if err != nil {
				log.Println(err)
			} else {

				format := video.Formats.WithAudioChannels() // Get matches with audio channels only

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

		if v.nowPlaying == (Song{}) {
			var err error
			v.voice, err = s.ChannelVoiceJoin(v.guildID, channelID, false, false)

			if err != nil {
				log.Println("ERROR: Error to join in a voice channel: ", err)
				return
			}

			// Bot joins caller's channel if it's not in it yet.
			if !alreadyInChannel {
				v.voice.Speaking(false)
				s.ChannelMessageSend(m.ChannelID, "**[Muse]** <@"+m.Author.ID+"> - I've joined your channel!")
			}

			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Playing ["+queue[0].Title+"] :notes:")
			playQueue(m)
		} else {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Queued... :infinity:")
			getQueue(m)
		}
	}
}

func stopSong(message string, m *discordgo.MessageCreate, v *VoiceInstance, channelId string) {
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Stopping ["+v.nowPlaying.Title+"] & Clearing Queue :octagonal_sign:")
	v.stop = true

	queue = []Song{}

	if v.encoder != nil {
		v.encoder.Cleanup()
	}

	v.voice.Disconnect()
}

func skipSong(message string, m *discordgo.MessageCreate, v *VoiceInstance, channelId string) {
	// Check if a song is playing - If no song, skip this and notify
	var replyMessage string
	if v.nowPlaying == (Song{}) {
		replyMessage = "**[Muse]** Queue is empty - There's nothing to skip!"
	} else {
		replyMessage = fmt.Sprintf("**[Muse]** Skipping [%s] :loop:", v.nowPlaying.Title)
		v.stop = true
		v.speaking = false

		if v.encoder != nil {
			v.encoder.Cleanup()
		}
		log.Println("Skipping " + v.nowPlaying.Title)
		log.Println("Queue Length: ", len(queue)-1)
	}
	s.ChannelMessageSend(m.ChannelID, replyMessage)
}

func getQueue(m *discordgo.MessageCreate) {
	queueList := ":musical_note:   QUEUE LIST   :musical_note:\n"
	if v.nowPlaying != (Song{}) {
		queueList = queueList + "Now Playing: " + v.nowPlaying.Title + "  ->  Queued by <@" + v.nowPlaying.User + "> \n"
	}
	for index, element := range queue {
		queueList = queueList + " " + strconv.Itoa(index+1) + ". " + element.Title + "  ->  Queued by <@" + element.User + "> \n"
	}

	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Fetching Queue...")
	s.ChannelMessageSend(m.ChannelID, queueList)
}

func removeFromQueue(message string, m *discordgo.MessageCreate) {
	// Split the message to get which song to remove from the queue
	commData := strings.Split(message, " ")
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
		v.stop = true

		// Nothing left in queue
		if len(queue) == 0 {
			v.nowPlaying = Song{}
			v.voice.Disconnect()
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Nothing left to play, peace! :v:")
		} else {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Next! Now playing ["+queue[0].Title+"] :loop:")
		}
	}
}
