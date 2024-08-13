package main

import (
	"encoding/json"
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
	botToken = os.Getenv("BOT_TOKEN")    // Set your discord bot token as an environment variable.
	youtubeToken = os.Getenv("YT_TOKEN") // Set your YouTube token as an environment variable.
	s, err = discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	service, err = youtube.NewService(ctx, option.WithAPIKey(youtubeToken))
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}

	setUpDcaOptions() // Encoder Settings

	// Read & store the list of bad quality songs
	file, _ := os.ReadFile("songQualityIssues.json")
	_ = json.Unmarshal([]byte(file), &badQualitySongs)

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
	log.Println("Session Open...")

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
	log.Println("Guild ID:", guildID)
	log.Println("Channel ID:", m.ChannelID)
	log.Println("Message:", m.Content)

	// Commands
	if m.Content != "" {
		if m.Content == "play help" {
			// TODO: Add Help Menu
		} else if m.Content == "play stuff" {
			go queueStuff(m)
		} else if m.Content == "play kudasai" {
			go queueKudasai(m)
		} else if strings.Contains(m.Content, "play") {
			go queueSong(m)
		} else if m.Content == "stop" {
			go stop(m)
		} else if strings.Contains(m.Content, "skip") {
			go skip(m)
		} else if m.Content == "queue" {
			go displayQueue(m)
		} else if strings.Contains(m.Content, "remove") {
			go remove(m)
		}
	} else {
		log.Println("Message is empty:", m.Content)
		return
	}
}
