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
	// Split the message to get YT link
	parsedContent := m.Content
	if strings.Contains(m.Content, "&index=") {
		parsedContent = strings.Split(m.Content, "&index=")[0]
	}

	if strings.Contains(m.Content, "&t=") {
		parsedContent = strings.Split(m.Content, "&t=")[0]
	}

	if strings.Contains(m.Content, "&start_radio") {
		parsedContent = strings.Split(m.Content, "&t=")[0]
	}

	commData := strings.Split(parsedContent, " ")

	queueLenBefore := len(queue)
	if commData[0] == "play" {
		if strings.Contains(m.Content, "https://www.youtube") {
			if strings.Contains(m.Content, "list") && strings.Contains(m.Content, "-pl") {
				playlistID := strings.SplitN(commData[2], "list=", 2)[1]
				if strings.Contains(playlistID, "PL-") {
					s.ChannelMessageSend(m.ChannelID, "**[Muse]** Queueing Your PlayList... :infinity:")
					queuePlaylist(playlistID, m)
				} else {
					s.ChannelMessageSend(m.ChannelID, "**[Muse]** Lists are not accepted, only playlists are. A valid link id contains PL :unamused:")
				}
			} else if strings.Contains(m.Content, "watch") && !strings.Contains(m.Content, "-pl") {

				link := commData[1]

				if strings.Contains(m.Content, "list") {
					link = strings.SplitN(commData[1], "list=", 2)[0]
				}

				getAndQueueSingleSong(m, link)
			}
			searchQueue = []SongSearch{}
			searchRequested = false
		} else {
			if len(commData) >= 2 {
				if input, err := strconv.Atoi(commData[1]); err == nil && searchRequested {
					if input <= len(searchQueue) {
						getAndQueueSingleSong(m, searchQueue[input-1].Id)
						searchQueue = []SongSearch{}
					}
					searchRequested = false
				} else if input, err := strconv.Atoi(commData[1]); err == nil && !searchRequested {
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
						skipSong(m)
					} else {
						s.ChannelMessageSend(m.ChannelID, "**[Muse]** Selected input was not in queue range")
					}
				} else {
					searchQueue = []SongSearch{}
					searchQuery := strings.SplitN(m.Content, "play ", 2)[1]
					getSearch(m, searchQueryList(searchQuery))
					searchRequested = true
				}
			}
		}

		// If there's nothing playing and the queue grew
		if v.nowPlaying == (Song{}) && len(queue) >= 1 {

			// Get the channel of the person who made the request
			authorChan := SearchVoiceChannel(m.Author.ID)

			// Join the channel of the person who made the request
			if authorChan != m.ChannelID {
				var err error
				v.voice, err = s.ChannelVoiceJoin(v.guildID, authorChan, true, true)
				if err != nil {
					if _, ok := s.VoiceConnections[v.guildID]; ok {
						v.voice = s.VoiceConnections[v.guildID]
					}
					log.Println("ERROR: Error to join in a voice channel: ", err)
					return
				}

				v.voice.Speaking(false)
				s.ChannelMessageSend(m.ChannelID, "**[Muse]** <@"+m.Author.ID+"> - I've joined your channel!")
			}

			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Playing ["+queue[0].Title+"] :notes:")
			playQueue(m)
		} else if !searchRequested {
			// Only display queue if it grew in size...
			if queueLenBefore < len(queue) {
				getQueue(m)
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
	}
}

// Stops current song and empties the queue
func stopAll(m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Stopping ["+v.nowPlaying.Title+"] & Clearing Queue :octagonal_sign:")
	v.stop = true
	searchRequested = false
	queue = []Song{}

	if v.encoder != nil {
		v.encoder.Cleanup()
	}

	v.voice.Disconnect()
}

// Skips the current song
func skipSong(m *discordgo.MessageCreate) {
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
	searchRequested = false
	s.ChannelMessageSend(m.ChannelID, replyMessage)
}

// Fetches and displays the queue
func getSearch(m *discordgo.MessageCreate, results map[string]string) {
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Fetching Search Results...")
	searchList := ":musical_note:   TOP RESULTS   :musical_note:\n"
	index := 1

	for id, name := range results {
		searchList = searchList + " " + strconv.Itoa(index) + ". " + name + "\n"
		index = index + 1
		songSearch = SongSearch{id, name}
		searchQueue = append(searchQueue, songSearch)
	}

	songSearch = SongSearch{}

	s.ChannelMessageSend(m.ChannelID, searchList)
	log.Println(searchList)
}
