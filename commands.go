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
	//TODO: Sanitize inputs on commData

	queueLenBefore := len(queue)
	if commDataIsValid {
		if strings.Contains(m.Content, "https://www.youtube") {
			if strings.Contains(m.Content, "list") && strings.Contains(m.Content, "-pl") || strings.Contains(m.Content, "/playlist?") {
				if strings.Contains(m.Content, "list=PL") {
					if len(commData) == 2 {
						if strings.Contains(commData[1], "list=") {
							println(m.Content)
							playlistID := strings.SplitN(commData[1], "list=", 2)[1]
							s.ChannelMessageSend(m.ChannelID, "**[Muse]** Queueing Your PlayList... :infinity:")
							queuePlaylist(playlistID, m)
						}
					} else if len(commData) == 3 {
						if strings.Contains(commData[2], "list=") {
							println(m.Content)
							playlistID := strings.SplitN(commData[2], "list=", 2)[1]
							s.ChannelMessageSend(m.ChannelID, "**[Muse]** Queueing Your PlayList... :infinity:")
							queuePlaylist(playlistID, m)
						}
					} else {
						s.ChannelMessageSend(m.ChannelID, "**[Muse]** The url must be the second or third parameter")
					}
				} else {
					s.ChannelMessageSend(m.ChannelID, "**[Muse]** Lists are not accepted, only playlists are. A valid link id contains PL :unamused:")
				}
			} else if strings.Contains(m.Content, "watch") && !strings.Contains(m.Content, "-pl") {
				if strings.Contains(commData[1], "list=") {
					link := strings.SplitN(commData[1], "list=", 2)[0]
					getAndQueueSingleSong(m, link)
				} else if strings.Contains(commData[1], "watch?") && !strings.Contains(commData[1], "list=") {
					link := commData[1]
					getAndQueueSingleSong(m, link)
				} else {
					s.ChannelMessageSend(m.ChannelID, "**[Muse]** The url must be the second or third parameter")
				}
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

func sanitizeQueueSongInputs(m *discordgo.MessageCreate) ([]string, bool) {
	parsedContent := m.Content
	isValid := false
	parsedContent = strings.Split(parsedContent, "&index=")[0]
	parsedContent = strings.Split(parsedContent, "&t=")[0]
	parsedContent = strings.Split(parsedContent, "&t=")[0]
	msgData := strings.Split(parsedContent, " ")

	if len(msgData) > 0 {
		var tmp []string
		commandPass := true
		selectionPass := true
		playlistPass := true
		playWasCalled := false

		// Remove any blank elements
		for _, value := range msgData {
			if value != " " && len(value) != 0 {
				if !playWasCalled && value == "play" {
					tmp = append(tmp, value)
					playWasCalled = true
				} else if playWasCalled && value != "play" {
					tmp = append(tmp, value)
				}
			}
		}
		msgData = tmp

		// First command MUST be play, this should always happen...
		if msgData[0] != "play" {
			commandPass = false
		}

		if len(msgData) == 1 {
			commandPass = false
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Insufficiant parameters!")
		}

		// If the input was numeric, it is assumed the user is selecting from the queue or search results
		if len(msgData) == 2 {
			if input, err := strconv.Atoi(msgData[1]); err == nil {
				if input <= 0 {
					selectionPass = false
					s.ChannelMessageSend(m.ChannelID, "**[Muse]** Your selection must be greater than 0")
				}
			}
		}

		// Check playlist input, it must always be the second option, and must
		// include a playlist in the link if selected.
		if len(msgData) >= 3 {
			if strings.Contains(parsedContent, " -pl ") {
				if msgData[1] == "-pl" {
					if strings.Contains(msgData[2], "youtube") {
						if !strings.Contains(msgData[2], "list=PL") {
							playlistPass = false
							s.ChannelMessageSend(m.ChannelID, "**[Muse]** You must enter a valid playlist, not a list - The ID must begin with PL.")
						}
					}
				} else {
					playlistPass = false
					s.ChannelMessageSend(m.ChannelID, "**[Muse]** When using the -pl parameter, it must be used immediately after play")
				}
			}
		}

		if commandPass && selectionPass && playlistPass {
			isValid = true
		}
	}

	return msgData, isValid
}
