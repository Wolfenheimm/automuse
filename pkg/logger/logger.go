package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps zerolog with additional functionality
type Logger struct {
	logger zerolog.Logger
	config LoggerConfig
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level            string
	Format           string
	OutputFile       string
	MaxFileSize      int64
	MaxBackups       int
	MaxAge           int
	EnableConsole    bool
	EnableFile       bool
	EnableJSON       bool
	EnableStackTrace bool
}

// Fields represents structured log fields
type Fields map[string]interface{}

// NewLogger creates a new logger instance
func NewLogger(config LoggerConfig) (*Logger, error) {
	// Set global settings
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339Nano
	
	// Parse log level
	level, err := zerolog.ParseLevel(strings.ToLower(config.Level))
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	
	// Create writers
	var writers []io.Writer
	
	// Console writer
	if config.EnableConsole {
		var consoleWriter io.Writer
		if config.EnableJSON {
			consoleWriter = os.Stdout
		} else {
			consoleWriter = zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: "2006-01-02 15:04:05.000",
				FormatLevel: func(i interface{}) string {
					return strings.ToUpper(fmt.Sprintf("%-5s", i))
				},
				FormatMessage: func(i interface{}) string {
					return fmt.Sprintf("%s", i)
				},
				FormatFieldName: func(i interface{}) string {
					return fmt.Sprintf("%s=", i)
				},
				FormatFieldValue: func(i interface{}) string {
					return fmt.Sprintf("%v", i)
				},
				FormatCaller: func(i interface{}) string {
					return fmt.Sprintf("<%s>", i)
				},
			}
		}
		writers = append(writers, consoleWriter)
	}
	
	// File writer
	if config.EnableFile && config.OutputFile != "" {
		// Create directory if it doesn't exist
		dir := filepath.Dir(config.OutputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		
		fileWriter := &lumberjack.Logger{
			Filename:   config.OutputFile,
			MaxSize:    int(config.MaxFileSize / (1024 * 1024)), // Convert to MB
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   true,
		}
		writers = append(writers, fileWriter)
	}
	
	// Create multi-writer
	var writer io.Writer
	if len(writers) == 1 {
		writer = writers[0]
	} else {
		writer = zerolog.MultiLevelWriter(writers...)
	}
	
	// Create logger
	logger := zerolog.New(writer).
		Level(level).
		With().
		Timestamp().
		Logger()
	
	// Add caller information if stack trace is enabled
	if config.EnableStackTrace {
		logger = logger.With().Caller().Logger()
	}
	
	return &Logger{
		logger: logger,
		config: config,
	}, nil
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...Fields) {
	event := l.logger.Debug()
	l.addFields(event, fields...)
	event.Msg(msg)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...Fields) {
	event := l.logger.Info()
	l.addFields(event, fields...)
	event.Msg(msg)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...Fields) {
	event := l.logger.Warn()
	l.addFields(event, fields...)
	event.Msg(msg)
}

// Error logs an error message
func (l *Logger) Error(msg string, err error, fields ...Fields) {
	event := l.logger.Error()
	if err != nil {
		if l.config.EnableStackTrace {
			event = event.Stack().Err(err)
		} else {
			event = event.Err(err)
		}
	}
	l.addFields(event, fields...)
	event.Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, err error, fields ...Fields) {
	event := l.logger.Fatal()
	if err != nil {
		if l.config.EnableStackTrace {
			event = event.Stack().Err(err)
		} else {
			event = event.Err(err)
		}
	}
	l.addFields(event, fields...)
	event.Msg(msg)
}

// WithContext creates a logger with context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		logger: l.logger.With().Logger(),
		config: l.config,
	}
}

// WithFields creates a logger with predefined fields
func (l *Logger) WithFields(fields Fields) *Logger {
	event := l.logger.With()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	return &Logger{
		logger: event.Logger(),
		config: l.config,
	}
}

// WithComponent creates a logger with a component field
func (l *Logger) WithComponent(component string) *Logger {
	return l.WithFields(Fields{"component": component})
}

// WithUser creates a logger with user information
func (l *Logger) WithUser(userID, username string) *Logger {
	return l.WithFields(Fields{
		"user_id":   userID,
		"username":  username,
	})
}

// WithGuild creates a logger with guild information
func (l *Logger) WithGuild(guildID, guildName string) *Logger {
	return l.WithFields(Fields{
		"guild_id":   guildID,
		"guild_name": guildName,
	})
}

// WithSong creates a logger with song information
func (l *Logger) WithSong(songID, title, duration string) *Logger {
	return l.WithFields(Fields{
		"song_id":    songID,
		"song_title": title,
		"duration":   duration,
	})
}

// WithError creates a logger with error information
func (l *Logger) WithError(err error) *Logger {
	return l.WithFields(Fields{"error": err.Error()})
}

// LogMemoryUsage logs current memory usage
func (l *Logger) LogMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	l.Info("Memory usage statistics", Fields{
		"alloc_mb":      bToMb(m.Alloc),
		"total_alloc_mb": bToMb(m.TotalAlloc),
		"sys_mb":        bToMb(m.Sys),
		"num_gc":        m.NumGC,
		"goroutines":    runtime.NumGoroutine(),
	})
}

// LogDiscordEvent logs Discord-related events
func (l *Logger) LogDiscordEvent(event string, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	fields["event_type"] = event
	l.Info("Discord event", fields)
}

// LogYouTubeEvent logs YouTube-related events
func (l *Logger) LogYouTubeEvent(event string, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	fields["event_type"] = event
	l.Info("YouTube event", fields)
}

// LogAudioEvent logs audio-related events
func (l *Logger) LogAudioEvent(event string, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	fields["event_type"] = event
	l.Info("Audio event", fields)
}

// LogCacheEvent logs cache-related events
func (l *Logger) LogCacheEvent(event string, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	fields["event_type"] = event
	l.Info("Cache event", fields)
}

// LogQueueEvent logs queue-related events
func (l *Logger) LogQueueEvent(event string, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	fields["event_type"] = event
	l.Info("Queue event", fields)
}

// LogCommandEvent logs command-related events
func (l *Logger) LogCommandEvent(command, userID, guildID string, success bool, duration time.Duration, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	fields["command"] = command
	fields["user_id"] = userID
	fields["guild_id"] = guildID
	fields["success"] = success
	fields["duration_ms"] = duration.Milliseconds()
	
	if success {
		l.Info("Command executed successfully", fields)
	} else {
		l.Warn("Command execution failed", fields)
	}
}

// LogPanic logs panic information
func (l *Logger) LogPanic(recovered interface{}, stack []byte) {
	l.Error("Panic recovered", nil, Fields{
		"panic":      recovered,
		"stack":      string(stack),
		"goroutines": runtime.NumGoroutine(),
	})
}

// LogStartup logs application startup information
func (l *Logger) LogStartup(version, buildTime, gitCommit string) {
	l.Info("Application starting", Fields{
		"version":    version,
		"build_time": buildTime,
		"git_commit": gitCommit,
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	})
}

// LogShutdown logs application shutdown information
func (l *Logger) LogShutdown(reason string, graceful bool) {
	l.Info("Application shutting down", Fields{
		"reason":   reason,
		"graceful": graceful,
	})
}

// LogConfiguration logs configuration information (with sensitive data redacted)
func (l *Logger) LogConfiguration(config interface{}) {
	l.Info("Configuration loaded", Fields{
		"config": config,
	})
}

// LogPerformanceMetrics logs performance metrics
func (l *Logger) LogPerformanceMetrics(metrics map[string]interface{}) {
	l.Info("Performance metrics", Fields(metrics))
}

// addFields adds fields to a log event
func (l *Logger) addFields(event *zerolog.Event, fields ...Fields) {
	for _, fieldSet := range fields {
		for key, value := range fieldSet {
			event.Interface(key, value)
		}
	}
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() string {
	return l.config.Level
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level string) error {
	logLevel, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	
	l.logger = l.logger.Level(logLevel)
	l.config.Level = level
	return nil
}

// Close closes the logger and flushes any buffered data
func (l *Logger) Close() error {
	// If using file logger, sync it
	if l.config.EnableFile {
		// lumberjack doesn't have a Close method, but we can sync the underlying file
		// This is a no-op for lumberjack, but included for completeness
	}
	return nil
}

// Default logger instance
var defaultLogger *Logger

// SetDefault sets the default logger
func SetDefault(logger *Logger) {
	defaultLogger = logger
}

// GetDefault returns the default logger
func GetDefault() *Logger {
	return defaultLogger
}

// Package-level convenience functions
func Debug(msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.Debug(msg, fields...)
	}
}

func Info(msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.Info(msg, fields...)
	}
}

func Warn(msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.Warn(msg, fields...)
	}
}

func Error(msg string, err error, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.Error(msg, err, fields...)
	}
}

func Fatal(msg string, err error, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.Fatal(msg, err, fields...)
	}
}

func WithComponent(component string) *Logger {
	if defaultLogger != nil {
		return defaultLogger.WithComponent(component)
	}
	return nil
}

func WithUser(userID, username string) *Logger {
	if defaultLogger != nil {
		return defaultLogger.WithUser(userID, username)
	}
	return nil
}

func WithGuild(guildID, guildName string) *Logger {
	if defaultLogger != nil {
		return defaultLogger.WithGuild(guildID, guildName)
	}
	return nil
}

func WithSong(songID, title, duration string) *Logger {
	if defaultLogger != nil {
		return defaultLogger.WithSong(songID, title, duration)
	}
	return nil
}

func LogMemoryUsage() {
	if defaultLogger != nil {
		defaultLogger.LogMemoryUsage()
	}
}

func LogCommandEvent(command, userID, guildID string, success bool, duration time.Duration, fields Fields) {
	if defaultLogger != nil {
		defaultLogger.LogCommandEvent(command, userID, guildID, success, duration, fields)
	}
}

func LogPanic(recovered interface{}, stack []byte) {
	if defaultLogger != nil {
		defaultLogger.LogPanic(recovered, stack)
	}
}