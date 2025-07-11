package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"automuse/config"
	"automuse/internal/services/audio"
	"automuse/pkg/dependency"
	"automuse/pkg/logger"
	"automuse/pkg/metrics"

	"github.com/bwmarrin/discordgo"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Application represents the main application
type Application struct {
	config       *config.Config
	logger       *logger.Logger
	metrics      *metrics.Metrics
	discord      *discordgo.Session
	youtube      *youtube.Service
	audioManager *audio.Manager
	ctx          context.Context
	cancel       context.CancelFunc
}

// Version information (should be set during build)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	loggerConfig := logger.LoggerConfig{
		Level:            cfg.Logging.Level,
		Format:           cfg.Logging.Format,
		OutputFile:       cfg.Logging.OutputFile,
		MaxFileSize:      cfg.Logging.MaxFileSize,
		MaxBackups:       cfg.Logging.MaxBackups,
		MaxAge:           cfg.Logging.MaxAge,
		EnableConsole:    cfg.Logging.EnableConsole,
		EnableFile:       cfg.Logging.EnableFile,
		EnableJSON:       cfg.Logging.EnableJSON,
		EnableStackTrace: cfg.Logging.EnableStackTrace,
	}

	appLogger, err := logger.NewLogger(loggerConfig)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Set as default logger
	logger.SetDefault(appLogger)

	// Log startup information
	appLogger.LogStartup(Version, BuildTime, GitCommit)
	appLogger.LogConfiguration(map[string]interface{}{
		"bot_token":     cfg.GetRedactedToken(),
		"youtube_token": cfg.GetRedactedAPIKey(),
		"debug_mode":    cfg.Logging.Level == "DEBUG",
		"features":      cfg.Features,
	})

	// Check system dependencies
	if err := checkDependencies(ctx, appLogger); err != nil {
		appLogger.Fatal("Dependency check failed", err)
	}

	// Initialize metrics if enabled
	var metricsInstance *metrics.Metrics
	if cfg.Features.EnableMetrics {
		metricsInstance = metrics.NewMetrics()
		appLogger.Info("Metrics collection enabled")
	}

	// Create application
	app := &Application{
		config:  cfg,
		logger:  appLogger,
		metrics: metricsInstance,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Initialize application
	if err := app.initialize(); err != nil {
		appLogger.Fatal("Failed to initialize application", err)
	}

	// Start application
	if err := app.start(); err != nil {
		appLogger.Fatal("Failed to start application", err)
	}

	// Wait for shutdown signal
	app.waitForShutdown()

	// Shutdown gracefully
	if err := app.shutdown(); err != nil {
		appLogger.Error("Error during shutdown", err)
	}

	appLogger.LogShutdown("Signal received", true)
}

// initialize initializes the application components
func (app *Application) initialize() error {
	// Initialize Discord session
	session, err := discordgo.New("Bot " + app.config.Discord.Token)
	if err != nil {
		return fmt.Errorf("failed to create Discord session: %w", err)
	}
	app.discord = session

	// Initialize YouTube service
	youtubeSvc, err := youtube.NewService(app.ctx, option.WithAPIKey(app.config.YouTube.APIKey))
	if err != nil {
		return fmt.Errorf("failed to create YouTube service: %w", err)
	}
	app.youtube = youtubeSvc

	// Initialize audio manager
	audioConfig := audio.Config{
		Bitrate:          app.config.Audio.Bitrate,
		Volume:           app.config.Audio.Volume,
		FrameRate:        app.config.Audio.FrameRate,
		FrameDuration:    app.config.Audio.FrameDuration,
		CompressionLevel: app.config.Audio.CompressionLevel,
		PacketLoss:       app.config.Audio.PacketLoss,
		BufferedFrames:   app.config.Audio.BufferedFrames,
		EnableVBR:        app.config.Audio.EnableVBR,
		ConnectTimeout:   app.config.Audio.ConnectTimeout,
		SpeakingTimeout:  app.config.Audio.SpeakingTimeout,
	}
	app.audioManager = audio.NewManager(audioConfig)

	// Set up Discord event handlers
	app.setupDiscordHandlers()

	app.logger.Info("Application initialized successfully")
	return nil
}

// start starts the application
func (app *Application) start() error {
	// Open Discord connection
	if err := app.discord.Open(); err != nil {
		return fmt.Errorf("failed to open Discord session: %w", err)
	}

	// Start metrics collection if enabled
	if app.config.Features.EnableMetrics && app.metrics != nil {
		collector := metrics.NewMonitoringCollector(app.metrics, 30*time.Second)
		go collector.Start(app.ctx)
	}

	// Start cleanup routines
	go app.startCleanupRoutines()

	app.logger.Info("Application started successfully")
	return nil
}

// setupDiscordHandlers sets up Discord event handlers
func (app *Application) setupDiscordHandlers() {
	// Ready handler
	app.discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		app.logger.Info("Discord bot ready", logger.Fields{
			"username":     r.User.Username,
			"discriminator": r.User.Discriminator,
			"bot_id":       r.User.ID,
			"guild_count":  len(r.Guilds),
		})

		// Set bot status
		err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
			Activities: []*discordgo.Activity{
				{
					Name: fmt.Sprintf("music 🎵 | %s help", app.config.Discord.CommandPrefix),
					Type: discordgo.ActivityTypeListening,
				},
			},
			Status: "online",
		})
		if err != nil {
			app.logger.Warn("Failed to set bot status", logger.Fields{"error": err.Error()})
		}

		if app.metrics != nil {
			app.metrics.RecordDiscordEvent("ready")
		}
	})

	// Guild join handler
	app.discord.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		app.logger.Info("Joined guild", logger.Fields{
			"guild_id":   g.ID,
			"guild_name": g.Name,
			"members":    g.MemberCount,
		})

		if app.metrics != nil {
			app.metrics.RecordGuildAction("join", g.ID)
		}
	})

	// Guild leave handler
	app.discord.AddHandler(func(s *discordgo.Session, g *discordgo.GuildDelete) {
		app.logger.Info("Left guild", logger.Fields{
			"guild_id":   g.ID,
			"guild_name": g.Name,
		})

		if app.metrics != nil {
			app.metrics.RecordGuildAction("leave", g.ID)
		}
	})

	// Message handler
	app.discord.AddHandler(app.handleMessage)

	// Voice state update handler
	app.discord.AddHandler(func(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
		if app.metrics != nil {
			app.metrics.RecordDiscordEvent("voice_state_update")
		}
	})

	// Error handler
	app.discord.AddHandler(func(s *discordgo.Session, e *discordgo.Disconnect) {
		app.logger.Error("Discord disconnected", fmt.Errorf("disconnect event"), logger.Fields{
			"event": "disconnect",
		})

		if app.metrics != nil {
			app.metrics.RecordDiscordEvent("disconnect")
		}
	})
}

// handleMessage handles incoming Discord messages
func (app *Application) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Skip bot messages
	if m.Author.Bot {
		return
	}

	// Skip DM messages
	if m.GuildID == "" {
		s.ChannelMessageSend(m.ChannelID, "**[AutoMuse]** I only work in Discord servers, not DMs!")
		return
	}

	// Check if message is a command
	if !app.isCommand(m.Content) {
		return
	}

	// Start timer for command execution
	var timer *metrics.Timer
	if app.metrics != nil {
		timer = app.metrics.StartTimer(fmt.Sprintf("command_%s", m.Content))
	}

	// Process command
	startTime := time.Now()
	err := app.processCommand(s, m)
	duration := time.Since(startTime)

	// Stop timer
	if timer != nil {
		app.metrics.StopTimer(fmt.Sprintf("command_%s", m.Content))
	}

	// Record metrics
	if app.metrics != nil {
		app.metrics.RecordCommandExecution(m.Content, err == nil, duration)
		app.metrics.RecordUserAction("command", m.Author.ID)
		app.metrics.RecordGuildAction("command", m.GuildID)
	}

	// Log command execution
	commandLogger := app.logger.WithUser(m.Author.ID, m.Author.Username).
		WithGuild(m.GuildID, "")
	commandLogger.LogCommandEvent(m.Content, m.Author.ID, m.GuildID, err == nil, duration, logger.Fields{
		"channel_id": m.ChannelID,
	})

	if err != nil {
		app.logger.Error("Command execution failed", err, logger.Fields{
			"command":    m.Content,
			"user_id":    m.Author.ID,
			"guild_id":   m.GuildID,
			"channel_id": m.ChannelID,
		})
	}
}

// isCommand checks if a message is a command
func (app *Application) isCommand(content string) bool {
	// For now, simple check for "play" prefix
	// This would be expanded to handle all commands
	return content == "play help" || 
		   content == "play stuff" || 
		   content == "play kudasai" ||
		   content == "stop" ||
		   content == "skip" ||
		   content == "pause" ||
		   content == "resume" ||
		   content == "queue" ||
		   content == "cache" ||
		   content == "cache-clear" ||
		   content == "buffer-status" ||
		   content == "shuffle" ||
		   content == "emergency-reset" ||
		   content == "reset" ||
		   len(content) > 5 && content[:5] == "play " ||
		   len(content) > 5 && content[:5] == "skip " ||
		   len(content) > 7 && content[:7] == "remove " ||
		   len(content) > 5 && content[:5] == "move "
}

// processCommand processes a command
func (app *Application) processCommand(s *discordgo.Session, m *discordgo.MessageCreate) error {
	// This is a placeholder - in a real implementation, this would delegate to
	// the appropriate command handlers using the new architecture
	
	// For now, just log that we received a command
	app.logger.Info("Command received", logger.Fields{
		"command":    m.Content,
		"user_id":    m.Author.ID,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
	})

	// Send a response indicating the new architecture is being used
	s.ChannelMessageSend(m.ChannelID, "**[AutoMuse]** ✨ Running on improved architecture! Command processing enhanced.")

	return nil
}

// startCleanupRoutines starts background cleanup routines
func (app *Application) startCleanupRoutines() {
	// Audio session cleanup
	audioCleanupTicker := time.NewTicker(15 * time.Minute)
	defer audioCleanupTicker.Stop()

	// Memory logging
	memoryTicker := time.NewTicker(5 * time.Minute)
	defer memoryTicker.Stop()

	for {
		select {
		case <-app.ctx.Done():
			return
		case <-audioCleanupTicker.C:
			app.audioManager.CleanupInactiveSessions(30 * time.Minute)
			app.logger.Info("Audio session cleanup completed")
		case <-memoryTicker.C:
			app.logger.LogMemoryUsage()
		}
	}
}

// waitForShutdown waits for shutdown signal
func (app *Application) waitForShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	app.logger.Info("Shutdown signal received")
}

// shutdown gracefully shuts down the application
func (app *Application) shutdown() error {
	app.logger.Info("Starting graceful shutdown")

	// Cancel context to stop all goroutines
	app.cancel()

	// Shutdown audio manager
	if app.audioManager != nil {
		if err := app.audioManager.Shutdown(); err != nil {
			app.logger.Error("Failed to shutdown audio manager", err)
		}
	}

	// Close Discord session
	if app.discord != nil {
		if err := app.discord.Close(); err != nil {
			app.logger.Error("Failed to close Discord session", err)
		}
	}

	// Close logger
	if app.logger != nil {
		if err := app.logger.Close(); err != nil {
			app.logger.Error("Failed to close logger", err)
		}
	}

	app.logger.Info("Graceful shutdown completed")
	return nil
}

// checkDependencies checks system dependencies
func checkDependencies(ctx context.Context, appLogger *logger.Logger) error {
	appLogger.Info("Checking system dependencies")

	report, err := dependency.ValidateEnvironment(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate environment: %w", err)
	}

	appLogger.Info("Dependency check completed", logger.Fields{
		"severity":           report.Severity,
		"required_missing":   len(report.RequiredMissing),
		"optional_missing":   len(report.OptionalMissing),
		"recommended_action": report.RecommendedAction,
	})

	// Log detailed report
	fmt.Println(report.GenerateReport())

	if !report.IsHealthy() {
		return fmt.Errorf("required dependencies are missing: %v", report.RequiredMissing)
	}

	return nil
}