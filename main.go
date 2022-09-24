package main

import (
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Initialize Discord & Setup Youtube
func init() {
	var err error
	botToken = os.Getenv("BOT_TOKEN_2") // Set your discord bot token as an environment variable.
	youtubeToken = os.Getenv("YT_TOKEN")
	s, err = discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	service, err = youtube.NewService(ctx, option.WithAPIKey(youtubeToken))
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}

	opts.RawOutput = true
	opts.Bitrate = 94
	opts.Application = "lowdelay"
	v.stop = true // Used to check if the bot is in channel playing music.
	searchRequested = false
}

func main() {
	// Add function handlers to trigger commands from discord chat
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Firing up...")
	})
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
	v.guildID = guildID
	v.session = s

	// Commands
	if m.Content != "" {
		if strings.Contains(m.Content, "play") {
			go queueSong(m)
		} else if m.Content == "stop" {
			go stopAll(m)
		} else if m.Content == "skip" {
			go skipSong(m)
		} else if m.Content == "queue" {
			go getQueue(m)
		} else if strings.Contains(m.Content, "remove") {
			go removeFromQueue(m)
		}
	} else {
		return
	}
}
