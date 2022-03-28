package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	yt "github.com/kkdai/youtube/v2"
)

// bot parameters
var (
	botToken       string
	voiceChannelID string
	s              *discordgo.Session
	o              chan string
	v              = new(VoiceInstance)
	client         = yt.Client{Debug: true}
	song           = Song{}
)

func init() {
	log.Println("In bot token")
	botToken = os.Getenv("BOT_TOKEN")
	flag.Parse()
	o = make(chan string)
}

// create discord session
func init() {
	log.Println("In discord init")
	var err error
	s, err = discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
	v.stop = true
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
	// avoid handling the message that the bot creates when replying to a user
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

	if message := extractor(m.Content); message != "" {
		if strings.Contains(message, "play") && strings.Contains(message, "youtube") {
			go playYT(message, m, v, voiceChannelID, channel)
		}
		if strings.Contains(message, "stop") {
			go stopYT(message, m, v, voiceChannelID)
		}
		if strings.Contains(message, "skip") {
			go skipYT(message, m, v, voiceChannelID)
		}
		if strings.Contains(message, "UwU") {
			go uwuYT(message, m, v, voiceChannelID, channel)
		}
	} else {
		return
	}
}

// Play Youtube Music in Channel
// Note: User must be in a voice channel for the bot to access said channel
func playYT(message string, m *discordgo.MessageCreate, v *VoiceInstance, channelId string, alreadyInChannel bool) {

	// Split the message to get YT link
	commData := strings.Split(message, " ")

	if len(commData) == 2 {
		var err error

		v.voice, err = s.ChannelVoiceJoin(v.guildID, channelId, false, false)

		if err != nil {
			log.Println("ERROR: Error to join in a voice channel: ", err)
			return
		}

		if !alreadyInChannel {
			v.voice.Speaking(false)
			s.ChannelMessageSend(m.ChannelID, "[**Bard**] <@"+m.Author.ID+"> - I've joined your channel!")
		}

		// Get Video Data
		video, err := client.GetVideo(commData[1])
		if err != nil {
			panic(err)
		}

		format := video.Formats.FindByQuality("medium") //TODO: Check if lower quality affects music quality

		song = Song{
			ChannelID: m.ChannelID,
			User:      m.Author.ID,
			ID:        m.ID,
			VidID:     video.ID,
			Title:     video.Title,
			Duration:  video.Duration.String(),
			VideoURL:  "",
		}

		if v.stop {
			s.ChannelMessageSend(m.ChannelID, "**[Player]** Playing "+song.Title+" :notes:")
		} else {
			s.ChannelMessageSend(m.ChannelID, "**[Player]** Queued "+song.Title+" :infinity:")
		}

		url, err := client.GetStreamURL(video, format)
		song.VideoURL = url

		v.nowPlaying = song
		v.stop = false
		v.speaking = true

		v.audioMutex.Lock()
		defer v.audioMutex.Unlock()

		v.voice.Speaking(true)
		v.DCA(song.VideoURL)
		v.stop = true
		v.speaking = false
	}
}

func stopYT(message string, m *discordgo.MessageCreate, v *VoiceInstance, channelId string) {
	s.ChannelMessageSend(m.ChannelID, "**[Player]** Stopping "+song.Title+" :octagonal_sign:")
	v.stop = true
	v.speaking = false

	if v.encoder != nil {
		v.encoder.Cleanup()
	}

}

func skipYT(message string, m *discordgo.MessageCreate, v *VoiceInstance, channelId string) {
	s.ChannelMessageSend(m.ChannelID, "**[Player]** Skipping "+song.Title+" :loop:")
	v.stop = true
	v.speaking = false

	if v.encoder != nil {
		v.encoder.Cleanup()
	}
}

func uwuYT(message string, m *discordgo.MessageCreate, v *VoiceInstance, channelId string, alreadyInChannel bool) {
	var err error

	v.voice, err = s.ChannelVoiceJoin(v.guildID, channelId, false, false)

	if err != nil {
		log.Println("ERROR: Error to join in a voice channel: ", err)
		return
	}

	if !alreadyInChannel {
		v.voice.Speaking(false)
		s.ChannelMessageSend(m.ChannelID, "[**Bard**] <@"+m.Author.ID+"> - I've joined your channel!")
	}

	// Get Video Data
	video, err := client.GetVideo("https://www.youtube.com/watch?v=rlkSMp7iz6c")
	if err != nil {
		panic(err)
	}

	format := video.Formats.FindByQuality("medium") //TODO: Check if lower quality affects music quality

	song = Song{
		ChannelID: m.ChannelID,
		User:      m.Author.ID,
		ID:        m.ID,
		VidID:     video.ID,
		Title:     video.Title,
		Duration:  video.Duration.String(),
		VideoURL:  "",
	}

	if v.stop {
		s.ChannelMessageSend(m.ChannelID, "**[Player]** UwU "+song.Title+" UwU")
	} else {
		s.ChannelMessageSend(m.ChannelID, "**[Player]** Queued UwU "+song.Title+" UwU")
	}

	url, err := client.GetStreamURL(video, format)
	song.VideoURL = url

	v.nowPlaying = song
	v.stop = false
	v.speaking = true

	v.audioMutex.Lock()
	defer v.audioMutex.Unlock()

	v.voice.Speaking(true)
	v.DCA(song.VideoURL)
	v.stop = true
	v.speaking = false
}

func extractor(content string) string {
	return content
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
