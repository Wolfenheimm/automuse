package main

import (
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
)

func sanitizeQueueSongInputs(m *discordgo.MessageCreate) ([]string, bool) {
	// Clean user input for later validation
	isValid := false
	parsedContent := m.Content
	parsedContent = strings.Split(parsedContent, "&index=")[0]
	parsedContent = strings.Split(parsedContent, "&t=")[0]
	msgData := strings.Split(parsedContent, " ")

	if len(msgData) > 0 {
		var tmp []string
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

		// The message data was empty - normally due to a user typing a word containing play
		if msgData == nil {
			return msgData, false
		}

		// First command MUST be play, this should always happen...
		if msgData[0] != "play" {
			return msgData, false
		}

		if len(msgData) == 1 {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Insufficiant parameters!")
			return msgData, false
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

		if selectionPass && playlistPass {
			isValid = true
		}
	}

	return msgData, isValid
}

func prepPlaylistCommand(commData []string, m *discordgo.MessageCreate) {
	// Only use lists starting with PL (Playlist only, lists are local to your own feed and cannot be used)
	if strings.Contains(m.Content, "list=PL") {
		// The url must be the second or third parameter
		if len(commData) == 2 {
			prepPlaylist(commData[1], m)
		} else if len(commData) == 3 {
			prepPlaylist(commData[2], m)
		} else {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** The url must be the second or third parameter")
		}
	} else {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Lists are not accepted, only playlists are. A valid link id contains PL :unamused:")
	}
}

func prepWatchCommand(commData []string, m *discordgo.MessageCreate) {
	if strings.Contains(commData[1], "list=") {
		queueSingleSong(m, strings.SplitN(commData[1], "list=", 2)[0])
	} else if strings.Contains(commData[1], "watch?") && !strings.Contains(commData[1], "list=") {
		queueSingleSong(m, commData[1])
	} else {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** The url must be the second or third parameter")
	}
}

func prepFirstSongEntered(m *discordgo.MessageCreate) {
	joinVoiceChannel(m)

	if len(queue) > 0 {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Playing ["+queue[0].Title+"] :notes:")
	}

	playQueue(m)
}

func prepSearchQueueSelector(commData []string, m *discordgo.MessageCreate) {
	if len(commData) >= 2 {
		if input, err := strconv.Atoi(commData[1]); err == nil && searchRequested {
			playFromSearch(input, m)
		} else if input, err := strconv.Atoi(commData[1]); err == nil && !searchRequested {
			playFromQueue(input, m)
		} else {
			getSearch(m)
		}
	}
}

func prepPlaylist(message string, m *discordgo.MessageCreate) {
	if strings.Contains(message, "list=") {
		println(m.Content)
		playlistID := strings.SplitN(message, "list=", 2)[1]
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Queueing Your PlayList... :infinity:")
		queuePlaylist(playlistID, m)
	}
}

func prepSongFormat(format youtube.FormatList, videoTitle string) *youtube.Format {
	// Select the correct video format - Check if it's in the song quality list file first
	formatList := &format[0]
	for _, value := range badQualitySongs.BadQualitySongNodes {
		if videoTitle == value.Title {
			if value.FormatNo < len(format) {
				formatList = &format[value.FormatNo]
			} else {
				log.Println("The format you set was not in range, using the first one instead.")
			}
			break
		}
	}

	return formatList
}

func prepSkip() {
	v.stop = true
	v.speaking = false
	if v.encoder != nil {
		v.encoder.Cleanup()
	}
}
