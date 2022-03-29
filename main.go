package main

import (
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	yt "github.com/kkdai/youtube/v2"
)

// Bot Parameters
var (
	botToken       string
	voiceChannelID string
	s              *discordgo.Session
	o              chan string
	v              = new(VoiceInstance)
	client         = yt.Client{Debug: true}
	song           = Song{}
	queue          = []Song{}
)

// Initialize Discord Session
func init() {
	var err error
	botToken = os.Getenv("BOT_TOKEN") // Set your discord bot token as an environment variable.
	s, err = discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
	v.stop = true // Used to check if the bot is in channel playing music.
}

func main() {
	// add function handlers for code execution
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) { log.Println("Bot is up!") })
	s.AddHandler(executionHandler)

	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Graceful shutdown")
}

func executionHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Avoid handling the message that the bot creates when replying to a user
	if m.Author.Bot {
		return
	}

	// Setup Channel Information
	guildID := SearchGuild(m.ChannelID)
	channel := false

	// Check if the request was made from a person in the same channel the bot is currently in
	if voiceChannelID == SearchVoiceChannel(m.Author.ID) {
		channel = true
	}

	voiceChannelID = SearchVoiceChannel(m.Author.ID)
	v.guildID = guildID
	v.session = s

	if m.Content != "" {
		if strings.Contains(m.Content, "play") && strings.Contains(m.Content, "youtube") {
			go queueYT(m.Content, m, v, voiceChannelID, channel)
			log.Printf("In queue")
		}
		if m.Content == "stop" {
			go stopYT(m.Content, m, v, voiceChannelID)
		}
		if m.Content == "skip" {
			go skipYT(m.Content, m, v, voiceChannelID)
		}
		if m.Content == "queue" {
			go getQueue(m)
		}
		if m.Content == "UwU" {
			go queueYT("play https://www.youtube.com/watch?v=rlkSMp7iz6c", m, v, voiceChannelID, channel)
		}
	} else {
		return
	}
}

// Play Youtube Music in Channel
// Note: User must be in a voice channel for the bot to access said channel
func queueYT(message string, m *discordgo.MessageCreate, v *VoiceInstance, channelId string, alreadyInChannel bool) {

	// Split the message to get YT link
	commData := strings.Split(message, " ")

	if len(commData) == 2 {
		var err error

		v.voice, err = s.ChannelVoiceJoin(v.guildID, channelId, false, false)

		if err != nil {
			log.Println("ERROR: Error to join in a voice channel: ", err)
			return
		}

		// Bot joins caller's channel if it's not in it yet.
		if !alreadyInChannel {
			v.voice.Speaking(false)
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** <@"+m.Author.ID+"> - I've joined your channel!")
		}

		// Get Video Data
		video, err := client.GetVideo(commData[1])
		if err != nil {
			panic(err)
		}

		format := video.Formats.FindByQuality("medium") //TODO: Check if lower quality affects music quality

		// Fill Song Info
		song = Song{
			ChannelID: m.ChannelID,
			User:      m.Author.ID,
			ID:        m.ID,
			VidID:     video.ID,
			Title:     video.Title,
			Duration:  video.Duration.String(),
			VideoURL:  "",
		}

		// Message to play or queue a song - v.stop used to see if a song is currently playing.
		if v.stop {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Playing ["+song.Title+"] :notes:")
		} else {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Queued ["+song.Title+"] :infinity:")
		}

		url, err := client.GetStreamURL(video, format)
		song.VideoURL = url
		queue = append(queue, song)

		// Check if a song is already playing, if not start playing the queue
		if v.nowPlaying == (Song{}) {
			log.Println("Next song was empty - playing songs")
			playQueue(m)
		} else {
			log.Println("Next song was not empty - song was queued - Playing: ", v.nowPlaying.Title)
		}
	}
}

func stopYT(message string, m *discordgo.MessageCreate, v *VoiceInstance, channelId string) {
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Stopping ["+v.nowPlaying.Title+"] & Clearing Queue :octagonal_sign:")
	v.stop = true

	queue = []Song{}

	if v.encoder != nil {
		v.encoder.Cleanup()
	}

	v.voice.Disconnect()
}

func skipYT(message string, m *discordgo.MessageCreate, v *VoiceInstance, channelId string) {
	// Check if a song is playing - If no song, skip this and notify
	if v.nowPlaying == (Song{}) {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Queue is empty - There's nothing to skip!")
	} else {
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Skipping ["+v.nowPlaying.Title+"] :loop:")
		v.stop = true
		v.speaking = false

		if v.encoder != nil {
			v.encoder.Cleanup()
		}
		log.Println("In Skip")
		log.Println("Queue Length: ", len(queue))
	}
}

func playQueue(m *discordgo.MessageCreate) {
	//v.audioMutex.Lock()
	//defer v.audioMutex.Unlock()
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
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Nothing left to play, peace fuckers! :v:")
		} else {
			s.ChannelMessageSend(m.ChannelID, "**[Muse]** Next! Now playing ["+queue[0].Title+"] :loop:")
		}
	}
}

func getQueue(m *discordgo.MessageCreate) {
	queueList := ":musical_note:   QUEUE LIST   :musical_note:\n"
	queueList = queueList + "Now Playing: " + v.nowPlaying.Title + "  ->  Queued by <@" + v.nowPlaying.User + "> \n"
	for index, element := range queue {
		queueList = queueList + " " + strconv.Itoa(index+1) + ". " + element.Title + "  ->  Queued by <@" + element.User + "> \n"
	}

	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Fetching Queue...")
	s.ChannelMessageSend(m.ChannelID, queueList)
}

func SearchGuild(textChannelID string) (guildID string) {
	channel, _ := s.Channel(textChannelID)
	guildID = channel.GuildID
	return
}

func SearchVoiceChannel(user string) (voiceChannelID string) {
	for _, g := range s.State.Guilds {
		for _, v := range g.VoiceStates {
			if v.UserID == user {
				return v.ChannelID
			}
		}
	}
	return ""
}

func (v *VoiceInstance) DCA(url string) {
	opts := dca.StdEncodeOptions
	opts.RawOutput = true
	opts.Bitrate = 96
	opts.Application = "lowdelay"

	encodeSession, err := dca.EncodeFile(url, opts)
	if err != nil {
		log.Println("FATA: Failed creating an encoding session: ", err)
	}
	v.encoder = encodeSession
	done := make(chan error)
	stream := dca.NewStream(encodeSession, v.voice, done)
	v.stream = stream
	for {
		select {
		case err := <-done:
			if err != nil && err != io.EOF {
				log.Println("FATA: An error occured", err)
			}
			// Clean up incase something happened and ffmpeg is still running
			encodeSession.Cleanup()
			return
		}
	}
}
