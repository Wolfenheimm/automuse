package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Configuration struct for server-agnostic operation
type Config struct {
	BotToken     string
	YoutubeToken string
	Debug        bool
}

// Global error handler
var errorHandler *ErrorHandler

// LoadConfig loads configuration from environment variables with validation
// Now only requires BOT_TOKEN and YT_TOKEN - server agnostic!
func LoadConfig() (*Config, error) {
	config := &Config{
		BotToken:     os.Getenv("BOT_TOKEN"),
		YoutubeToken: os.Getenv("YT_TOKEN"),
		Debug:        os.Getenv("DEBUG") == "true",
	}

	// Validate required configuration (only tokens needed now)
	var missing []string
	if config.BotToken == "" {
		missing = append(missing, "BOT_TOKEN")
	}
	if config.YoutubeToken == "" {
		missing = append(missing, "YT_TOKEN")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	// Log configuration (with sensitive data redacted)
	log.Printf("Configuration loaded:")
	log.Printf("- Bot Token: %s***", config.BotToken[:8])
	log.Printf("- YouTube Token: %s***", config.YoutubeToken[:8])
	log.Printf("- Debug Mode: %t", config.Debug)
	log.Printf("- Server Agnostic: ✅ (will work in any server)")

	return config, nil
}

// Initialize Discord & Setup Youtube
func init() {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	botToken = config.BotToken
	youtubeToken = config.YoutubeToken

	// Initialize Discord session
	s, err = discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	// Initialize error handler
	errorHandler = NewErrorHandler(s)

	// Initialize YouTube service
	service, err = youtube.NewService(ctx, option.WithAPIKey(youtubeToken))
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}

	// Initialize metadata manager
	metadataManager = NewMetadataManager("downloads/metadata.json")

	// Cleanup any missing files on startup
	if err := metadataManager.CleanupMissing(); err != nil {
		log.Printf("WARN: Failed to cleanup missing files: %v", err)
	}

	v.stop = true // Used to check if the bot is in channel playing music.
	searchRequested = false

	log.Println("AutoMuse initialization completed successfully")
}

func main() {
	log.Println("Starting AutoMuse Discord Music Bot...")

	// Add function handlers to trigger commands from discord chat
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Bot is ready! Logged in as: %s#%s", r.User.Username, r.User.Discriminator)
		log.Printf("Bot ID: %s", r.User.ID)
		log.Printf("Connected to %d guilds", len(r.Guilds))

		// Set bot status
		err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
			Activities: []*discordgo.Activity{
				{
					Name: "music 🎵 | play help",
					Type: discordgo.ActivityTypeListening,
				},
			},
			Status: "online",
		})
		if err != nil {
			log.Printf("WARN: Failed to set bot status: %v", err)
		}
	})

	s.AddHandler(executionHandler)

	// Open Discord session
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer func() {
		log.Println("Closing Discord session...")
		s.Close()
	}()

	log.Println("Session Open - AutoMuse is now running!")
	log.Println("Press Ctrl+C to exit")

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-stop

	log.Println("Shutdown signal received, cleaning up...")

	// Graceful shutdown procedures
	if v.voice != nil {
		log.Println("Disconnecting from voice channel...")
		v.voice.Disconnect()
	}

	if bufferManager != nil {
		log.Println("Stopping buffer manager...")
		bufferManager.StopBuffering()
	}

	if metadataManager != nil {
		log.Println("Saving metadata...")
		metadataManager.SaveMetadata()
	}

	log.Println("AutoMuse shutdown complete")
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
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		showHelp(m)
	}()
	return nil
}

type PlayStuffCommand struct{}

func (p *PlayStuffCommand) CanHandle(content string) bool {
	return content == "play stuff"
}
func (p *PlayStuffCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		queueStuff(m)
	}()
	return nil
}

type PlayKudasaiCommand struct{}

func (p *PlayKudasaiCommand) CanHandle(content string) bool {
	return content == "play kudasai"
}
func (p *PlayKudasaiCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		queueKudasai(m)
	}()
	return nil
}

type PlayCommand struct{}

func (p *PlayCommand) CanHandle(content string) bool {
	return strings.Contains(content, "play") && content != "play help" && content != "play stuff" && content != "play kudasai"
}
func (p *PlayCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	// Validate input before processing
	content := strings.TrimSpace(m.Content)
	if len(content) < 5 { // "play" + space + at least 1 char
		return NewValidationError("Play command requires additional parameters", nil).
			WithContext("command", m.Content).
			WithContext("user_id", m.Author.ID)
	}

	// Additional validation for content safety
	if strings.Contains(content, "\n") || strings.Contains(content, "\r") {
		return NewValidationError("Invalid characters in command", nil).
			WithContext("command", m.Content).
			WithContext("user_id", m.Author.ID)
	}

	// Check for excessively long commands
	if len(content) > 2000 {
		return NewValidationError("Command too long", nil).
			WithContext("command_length", len(content)).
			WithContext("user_id", m.Author.ID)
	}

	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		queueSong(m)
	}()
	return nil
}

type StopCommand struct{}

func (s *StopCommand) CanHandle(content string) bool {
	return content == "stop"
}
func (s *StopCommand) Handle(sess *discordgo.Session, m *discordgo.MessageCreate) error {
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		stop(m)
	}()
	return nil
}

type SkipCommand struct{}

func (s *SkipCommand) CanHandle(content string) bool {
	return strings.Contains(content, "skip")
}
func (s *SkipCommand) Handle(sess *discordgo.Session, m *discordgo.MessageCreate) error {
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		skip(m)
	}()
	return nil
}

type QueueCommand struct{}

func (q *QueueCommand) CanHandle(content string) bool {
	return content == "queue"
}
func (q *QueueCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		displayQueue(m)
	}()
	return nil
}

type RemoveCommand struct{}

func (r *RemoveCommand) CanHandle(content string) bool {
	return strings.Contains(content, "remove")
}
func (r *RemoveCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	// Validate remove command format
	parts := strings.Fields(m.Content)
	if len(parts) != 2 {
		return NewValidationError("Remove command requires a position number", nil).
			WithContext("command", m.Content).
			WithContext("user_id", m.Author.ID)
	}

	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		remove(m)
	}()
	return nil
}

type CacheCommand struct{}

func (c *CacheCommand) CanHandle(content string) bool {
	return content == "cache"
}
func (c *CacheCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		cacheStatsCommand(s, m, nil)
	}()
	return nil
}

type CacheClearCommand struct{}

func (c *CacheClearCommand) CanHandle(content string) bool {
	return content == "cache-clear"
}
func (c *CacheClearCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		cacheClearCommand(s, m, nil)
	}()
	return nil
}

type BufferStatusCommand struct{}

func (b *BufferStatusCommand) CanHandle(content string) bool {
	return content == "buffer-status"
}
func (b *BufferStatusCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		bufferStatusCommand(s, m, nil)
	}()
	return nil
}

type MoveQueueCommand struct{}

func (mq *MoveQueueCommand) CanHandle(content string) bool {
	return strings.HasPrefix(content, "move ")
}
func (mq *MoveQueueCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	parts := strings.Fields(m.Content)
	if len(parts) < 3 {
		return fmt.Errorf("move command requires from and to positions")
	}

	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		moveQueueCommand(s, m, parts[1:])
	}()
	return nil
}

type ShuffleQueueCommand struct{}

func (sq *ShuffleQueueCommand) CanHandle(content string) bool {
	return content == "shuffle"
}
func (sq *ShuffleQueueCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		shuffleQueueCommand(s, m, nil)
	}()
	return nil
}

type EmergencyResetCommand struct{}

func (er *EmergencyResetCommand) CanHandle(content string) bool {
	return content == "emergency-reset" || content == "reset"
}
func (er *EmergencyResetCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	// Only allow certain users to use emergency reset (you can modify this check)
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		emergencyResetCommand(s, m)
	}()
	return nil
}

type PauseCommand struct{}

func (p *PauseCommand) CanHandle(content string) bool {
	return content == "pause"
}
func (p *PauseCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		pauseCommand(s, m)
	}()
	return nil
}

type ResumeCommand struct{}

func (r *ResumeCommand) CanHandle(content string) bool {
	return content == "resume"
}
func (r *ResumeCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate) error {
	go func() {
		defer RecoverWithErrorHandler(errorHandler, m.ChannelID)
		resumeCommand(s, m)
	}()
	return nil
}

var commandHandlers = []CommandHandler{
	&PlayHelpCommand{}, // Check specific play commands first
	&PlayStuffCommand{},
	&PlayKudasaiCommand{},
	&PlayCommand{}, // General play command last
	&StopCommand{},
	&SkipCommand{},
	&PauseCommand{},  // Add pause command
	&ResumeCommand{}, // Add resume command
	&QueueCommand{},
	&RemoveCommand{},
	&CacheCommand{},
	&CacheClearCommand{},
	&BufferStatusCommand{},
	&MoveQueueCommand{},
	&ShuffleQueueCommand{},
	&EmergencyResetCommand{}, // Add emergency reset command
}

func executionHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Avoid handling the message that the bot creates when replying to a user
	if m.Author.Bot {
		return
	}

	// Server-agnostic setup - dynamically get guild info from the message
	guildID := m.GuildID // Use the guild where the message was sent
	if guildID == "" {
		// Handle DMs gracefully
		log.Printf("Command received in DM from %s: %s", m.Author.Username, m.Content)
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** I only work in Discord servers, not DMs!")
		return
	}

	// Dynamic voice instance setup per guild/session
	v.guildID = guildID
	v.session = s

	log.Printf("Processing command from Guild: %s, Channel: %s, User: %s, Message: %s",
		guildID, m.ChannelID, m.Author.Username, m.Content)

	// Handle commands using the new pattern
	for _, handler := range commandHandlers {
		if handler.CanHandle(m.Content) {
			if err := handler.Handle(s, m); err != nil {
				// Use structured error handling
				errorHandler.Handle(err, m.ChannelID)
			}
			return
		}
	}

	// If no command was recognized and message is not empty, log it (optional)
	if m.Content != "" && strings.HasPrefix(m.Content, "play") {
		log.Printf("Unrecognized play command variant: %s", m.Content)
		s.ChannelMessageSend(m.ChannelID, "**[Muse]** Command not recognized. Try `play help` for available commands.")
	}
}
