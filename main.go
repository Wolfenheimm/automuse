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

// CommandHandler interface for better command organization
type CommandHandler interface {
	Handle(s *discordgo.Session, m *discordgo.MessageCreate) error
	CanHandle(content string) bool
}

type PlayHelpCommand struct{}

func (p *PlayHelpCommand) CanHandle(content string) bool {
	return content == "play help"
}
func (p *PlayHelpCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go showHelp(m)
	return nil
}

type PlayStuffCommand struct{}

func (p *PlayStuffCommand) CanHandle(content string) bool {
	return content == "play stuff"
}
func (p *PlayStuffCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go queueStuff(m)
	return nil
}

type PlayKudasaiCommand struct{}

func (p *PlayKudasaiCommand) CanHandle(content string) bool {
	return content == "play kudasai"
}
func (p *PlayKudasaiCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go queueKudasai(m)
	return nil
}

type PlayCommand struct{}

func (p *PlayCommand) CanHandle(content string) bool {
	return strings.Contains(content, "play") && content != "play help" && content != "play stuff" && content != "play kudasai"
}
func (p *PlayCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go queueSong(m)
	return nil
}

type StopCommand struct{}

func (s *StopCommand) CanHandle(content string) bool {
	return content == "stop"
}
func (s *StopCommand) Handle(sess *discordgo.Session, m *discordgo.MessageCreate) error {
	go stop(m)
	return nil
}

type SkipCommand struct{}

func (s *SkipCommand) CanHandle(content string) bool {
	return strings.Contains(content, "skip")
}
func (s *SkipCommand) Handle(sess *discordgo.Session, m *discordgo.MessageCreate) error {
	go skip(m)
	return nil
}

type QueueCommand struct{}

func (q *QueueCommand) CanHandle(content string) bool {
	return content == "queue"
}
func (q *QueueCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go displayQueue(m)
	return nil
}

type RemoveCommand struct{}

func (r *RemoveCommand) CanHandle(content string) bool {
	return strings.Contains(content, "remove")
}
func (r *RemoveCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go remove(m)
	return nil
}

var commandHandlers = []CommandHandler{
	&PlayHelpCommand{}, // Check specific play commands first
	&PlayStuffCommand{},
	&PlayKudasaiCommand{},
	&PlayCommand{}, // General play command last
	&StopCommand{},
	&SkipCommand{},
	&QueueCommand{},
	&RemoveCommand{},
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
	log.Printf("Processing command from Guild: %s, Channel: %s, Message: %s",
		guildID, m.ChannelID, m.Content)

	// Handle commands using the new pattern
	for _, handler := range commandHandlers {
		if handler.CanHandle(m.Content) {
			if err := handler.Handle(s, m); err != nil {
				log.Printf("ERROR: Command handling failed: %v", err)
				s.ChannelMessageSend(m.ChannelID, "**[Muse]** Command failed. Please try again.")
			}
			return
		}
	}

	// If no command was recognized and message is not empty, log it
	if m.Content != "" {
		log.Printf("Unrecognized command: %s", m.Content)
	}
}
