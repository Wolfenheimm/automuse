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

	// If the message data is not empty, check if the user entered a valid command
	if len(msgData) > 0 {
		var tmp []string
		selectionPass := true
		playlistPass := true
		playWasCalled := false

		// Remove any blank elements from the message data
		for _, value := range msgData {
			// If the value is not a space or empty, append it to the temporary slice
			if value != " " && len(value) != 0 {
				// If the value is play, and it has not been called yet, append it
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

		// If the user only entered play, it is assumed they were simply saying the word...
		if len(msgData) == 1 {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Insufficient parameters!")
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

		// If both selection and playlist pass, the input is valid
		if selectionPass && playlistPass {
			isValid = true
		}
	}

	return msgData, isValid
}

// Prepares the play command if a playlist is found in the url
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

// Checks if the user is queuing a song or playlist
func prepWatchCommand(commData []string, m *discordgo.MessageCreate) bool {
	// Check if this is a playlist URL (contains list= or /playlist?)
	if strings.Contains(commData[1], "list=") || strings.Contains(commData[1], "/playlist?") {
		log.Printf("INFO: Detected playlist URL: %s", commData[1])

		var playlistID string
		
		// Extract playlist ID from different URL formats
		if strings.Contains(commData[1], "/playlist?list=") {
			// Format: https://www.youtube.com/playlist?list=PL...
			parts := strings.SplitN(commData[1], "list=", 2)
			if len(parts) > 1 {
				playlistID = strings.Split(parts[1], "&")[0]
			}
		} else if strings.Contains(commData[1], "list=") {
			// Format: https://www.youtube.com/watch?v=...&list=PL...
			parts := strings.SplitN(commData[1], "list=", 2)
			if len(parts) > 1 {
				playlistID = strings.Split(parts[1], "&")[0]
			}
		}

		if playlistID != "" {
			log.Printf("INFO: Extracted playlist ID: %s", playlistID)
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Detected playlist! Starting first song and queuing rest in background... :infinity:")

			// Start the threaded playlist processing
			queuePlaylistThreaded(playlistID, m)
			return true // Indicate that playback was already started
		} else {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Could not extract playlist ID from URL")
		}
	} else if strings.Contains(commData[1], "watch?") {
		// Regular single video (without playlist)
		queueSingleSong(m, commData[1])
	} else {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** The url must be the second or third parameter")
	}
	return false // Regular queueing, no playback started yet
}

// Prepares the play command when a song is manually entered
func prepFirstSongEntered(m *discordgo.MessageCreate, isManual bool) {
	// **CRITICAL PLAYBACK PROTECTION** - Prevent multiple simultaneous playback
	if getPlaybackState() {
		log.Printf("WARN: Playback already in progress, rejecting new playback request")
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** 🚫 Playback is already in progress. Use `skip` or `stop` to control current playback.")
		return
	}

	// Atomically set playback state
	setPlaybackState(true)
	defer setPlaybackState(false)

	if len(queue) > 0 {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Playing ["+queue[0].Title+"] :notes:")
	}

	if isManual {
		playQueue(m, true)
	} else {
		playQueue(m, false)
	}
}

// Prepares the play command when a numerical option is chosen (queue or search)
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

// Prepares the songs to be queued from the playlist
func prepPlaylist(message string, m *discordgo.MessageCreate) {
	// Check if the message contains a playlist by checking the list= parameter
	if strings.Contains(message, "list=") {
		println(m.Content)
		playlistID := strings.SplitN(message, "list=", 2)[1]
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Queueing Your PlayList... :infinity:")
		queuePlaylist(playlistID, m)
	}
}

// Prepares the song's format for playback (quality)
func prepSongFormat(format youtube.FormatList) *youtube.Format {
	// First try to find formats without cipher requirements
	for _, f := range format {
		// Check if the format has a direct URL (no cipher)
		if f.URL != "" && f.Cipher == "" {
			log.Printf("[DEBUG] Found format without cipher: Itag=%d, Quality=%s, MimeType=%s",
				f.ItagNo, f.Quality, f.MimeType)
			return &f
		}
	}

	// If no direct URL formats found, try audio-only formats
	for _, f := range format {
		if f.ItagNo == 140 || f.ItagNo == 251 || f.ItagNo == 250 || f.ItagNo == 249 {
			log.Printf("[DEBUG] Selected audio-only format: Itag=%d, Quality=%s, MimeType=%s",
				f.ItagNo, f.Quality, f.MimeType)
			return &f
		}
	}

	// If no suitable format found, fall back to the first format
	log.Printf("[DEBUG] No suitable format found, falling back to first format")
	return &format[0]
}

// Preps the skip command
func prepSkip() {
	log.Printf("INFO: Skip command initiated")
	v.stop = true
	v.speaking = false
	v.paused = false // Reset pause state when skipping

	// Force stop the current audio stream
	if v.voice != nil {
		v.voice.Speaking(false)
		log.Printf("INFO: Set speaking to false for skip")
	}

	// Cleanup encoder if it exists
	if v.encoder != nil {
		log.Printf("INFO: Cleaning up encoder for skip")
		v.encoder.Cleanup()
		v.encoder = nil
	}

	// Cleanup stream if it exists
	if v.stream != nil {
		log.Printf("INFO: Cleaning up stream for skip")
		v.stream = nil
	}

	log.Printf("INFO: Skip preparation completed")
}
