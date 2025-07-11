package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	Discord  DiscordConfig  `json:"discord"`
	YouTube  YouTubeConfig  `json:"youtube"`
	Audio    AudioConfig    `json:"audio"`
	Queue    QueueConfig    `json:"queue"`
	Cache    CacheConfig    `json:"cache"`
	Logging  LoggingConfig  `json:"logging"`
	Features FeatureConfig  `json:"features"`
}

// DiscordConfig holds Discord-specific configuration
type DiscordConfig struct {
	Token              string        `json:"token"`
	CommandPrefix      string        `json:"command_prefix"`
	MaxMessageLength   int           `json:"max_message_length"`
	ReconnectAttempts  int           `json:"reconnect_attempts"`
	ReconnectDelay     time.Duration `json:"reconnect_delay"`
	ShardCount         int           `json:"shard_count"`
	EnableSlashCmds    bool          `json:"enable_slash_commands"`
}

// YouTubeConfig holds YouTube API configuration
type YouTubeConfig struct {
	APIKey            string        `json:"api_key"`
	MaxSearchResults  int           `json:"max_search_results"`
	RequestTimeout    time.Duration `json:"request_timeout"`
	RetryAttempts     int           `json:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay"`
	EnableFallback    bool          `json:"enable_fallback"`
	FallbackMethod    string        `json:"fallback_method"`
}

// AudioConfig holds audio processing configuration
type AudioConfig struct {
	Bitrate            int           `json:"bitrate"`
	Volume             int           `json:"volume"`
	FrameRate          int           `json:"frame_rate"`
	FrameDuration      int           `json:"frame_duration"`
	CompressionLevel   int           `json:"compression_level"`
	PacketLoss         int           `json:"packet_loss"`
	BufferedFrames     int           `json:"buffered_frames"`
	MaxBufferedFrames  int           `json:"max_buffered_frames"`
	MinBufferedFrames  int           `json:"min_buffered_frames"`
	EnableVBR          bool          `json:"enable_vbr"`
	ConnectTimeout     time.Duration `json:"connect_timeout"`
	SpeakingTimeout    time.Duration `json:"speaking_timeout"`
}

// QueueConfig holds queue management configuration
type QueueConfig struct {
	MaxSize               int           `json:"max_size"`
	MaxPlaylistSize       int           `json:"max_playlist_size"`
	PlaylistCooldown      time.Duration `json:"playlist_cooldown"`
	MaxConcurrentPlaylists int          `json:"max_concurrent_playlists"`
	UserRateLimitDelay    time.Duration `json:"user_rate_limit_delay"`
	CommandTimeoutDelay   time.Duration `json:"command_timeout_delay"`
	ShuffleAlgorithm      string        `json:"shuffle_algorithm"`
	EnableDuplicateCheck  bool          `json:"enable_duplicate_check"`
}

// CacheConfig holds caching configuration
type CacheConfig struct {
	CacheDirectory    string        `json:"cache_directory"`
	MaxCacheSize      int64         `json:"max_cache_size"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	MaxFileAge        time.Duration `json:"max_file_age"`
	BufferSize        int           `json:"buffer_size"`
	EnablePredownload bool          `json:"enable_predownload"`
	CompressionLevel  int           `json:"compression_level"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level            string `json:"level"`
	Format           string `json:"format"`
	OutputFile       string `json:"output_file"`
	MaxFileSize      int64  `json:"max_file_size"`
	MaxBackups       int    `json:"max_backups"`
	MaxAge           int    `json:"max_age"`
	EnableConsole    bool   `json:"enable_console"`
	EnableFile       bool   `json:"enable_file"`
	EnableJSON       bool   `json:"enable_json"`
	EnableStackTrace bool   `json:"enable_stack_trace"`
}

// FeatureConfig holds feature flags
type FeatureConfig struct {
	EnableCaching       bool     `json:"enable_caching"`
	EnableBuffering     bool     `json:"enable_buffering"`
	EnableMetrics       bool     `json:"enable_metrics"`
	EnableRateLimiting  bool     `json:"enable_rate_limiting"`
	EnableAutoReconnect bool     `json:"enable_auto_reconnect"`
	EnableAdvancedAudio bool     `json:"enable_advanced_audio"`
	SupportedFormats    []string `json:"supported_formats"`
	ExperimentalFeatures []string `json:"experimental_features"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Discord: DiscordConfig{
			Token:              "",
			CommandPrefix:      "play",
			MaxMessageLength:   1000,
			ReconnectAttempts:  5,
			ReconnectDelay:     2 * time.Second,
			ShardCount:         1,
			EnableSlashCmds:    false,
		},
		YouTube: YouTubeConfig{
			APIKey:            "",
			MaxSearchResults:  10,
			RequestTimeout:    30 * time.Second,
			RetryAttempts:     3,
			RetryDelay:        1 * time.Second,
			EnableFallback:    true,
			FallbackMethod:    "yt-dlp",
		},
		Audio: AudioConfig{
			Bitrate:            128,
			Volume:             256,
			FrameRate:          48000,
			FrameDuration:      20,
			CompressionLevel:   5,
			PacketLoss:         1,
			BufferedFrames:     200,
			MaxBufferedFrames:  500,
			MinBufferedFrames:  100,
			EnableVBR:          true,
			ConnectTimeout:     10 * time.Second,
			SpeakingTimeout:    5 * time.Second,
		},
		Queue: QueueConfig{
			MaxSize:               500,
			MaxPlaylistSize:       100,
			PlaylistCooldown:      5 * time.Second,
			MaxConcurrentPlaylists: 3,
			UserRateLimitDelay:    3 * time.Second,
			CommandTimeoutDelay:   2 * time.Second,
			ShuffleAlgorithm:      "fisher-yates",
			EnableDuplicateCheck:  true,
		},
		Cache: CacheConfig{
			CacheDirectory:    "downloads",
			MaxCacheSize:      10 * 1024 * 1024 * 1024, // 10GB
			CleanupInterval:   24 * time.Hour,
			MaxFileAge:        7 * 24 * time.Hour,
			BufferSize:        5,
			EnablePredownload: true,
			CompressionLevel:  6,
		},
		Logging: LoggingConfig{
			Level:            "INFO",
			Format:           "structured",
			OutputFile:       "logs/automuse.log",
			MaxFileSize:      100 * 1024 * 1024, // 100MB
			MaxBackups:       5,
			MaxAge:           30,
			EnableConsole:    true,
			EnableFile:       true,
			EnableJSON:       false,
			EnableStackTrace: true,
		},
		Features: FeatureConfig{
			EnableCaching:       true,
			EnableBuffering:     true,
			EnableMetrics:       false,
			EnableRateLimiting:  true,
			EnableAutoReconnect: true,
			EnableAdvancedAudio: true,
			SupportedFormats:    []string{"mp3", "m4a", "webm", "ogg"},
			ExperimentalFeatures: []string{},
		},
	}
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := DefaultConfig()

	// Load Discord configuration
	if token := os.Getenv("BOT_TOKEN"); token != "" {
		config.Discord.Token = token
	}

	// Load YouTube configuration
	if apiKey := os.Getenv("YT_TOKEN"); apiKey != "" {
		config.YouTube.APIKey = apiKey
	}

	// Load debug mode
	if debug := os.Getenv("DEBUG"); debug == "true" {
		config.Logging.Level = "DEBUG"
	}

	// Load optional environment variables with defaults
	if maxQueue := os.Getenv("MAX_QUEUE_SIZE"); maxQueue != "" {
		if size, err := strconv.Atoi(maxQueue); err == nil {
			config.Queue.MaxSize = size
		}
	}

	if maxPlaylistSize := os.Getenv("MAX_PLAYLIST_SIZE"); maxPlaylistSize != "" {
		if size, err := strconv.Atoi(maxPlaylistSize); err == nil {
			config.Queue.MaxPlaylistSize = size
		}
	}

	if cacheDir := os.Getenv("CACHE_DIR"); cacheDir != "" {
		config.Cache.CacheDirectory = cacheDir
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = strings.ToUpper(logLevel)
	}

	if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		config.Logging.OutputFile = logFile
	}

	// Feature flags
	if enableCaching := os.Getenv("ENABLE_CACHING"); enableCaching == "false" {
		config.Features.EnableCaching = false
	}

	if enableBuffering := os.Getenv("ENABLE_BUFFERING"); enableBuffering == "false" {
		config.Features.EnableBuffering = false
	}

	if enableMetrics := os.Getenv("ENABLE_METRICS"); enableMetrics == "true" {
		config.Features.EnableMetrics = true
	}

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	var errors []string

	// Validate Discord configuration
	if c.Discord.Token == "" {
		errors = append(errors, "Discord token (BOT_TOKEN) is required")
	}

	// Validate YouTube configuration
	if c.YouTube.APIKey == "" {
		errors = append(errors, "YouTube API key (YT_TOKEN) is required")
	}

	// Validate audio configuration
	if c.Audio.Bitrate < 32 || c.Audio.Bitrate > 320 {
		errors = append(errors, "audio bitrate must be between 32 and 320 kbps")
	}

	if c.Audio.Volume < 0 || c.Audio.Volume > 1024 {
		errors = append(errors, "audio volume must be between 0 and 1024")
	}

	if c.Audio.BufferedFrames < c.Audio.MinBufferedFrames || c.Audio.BufferedFrames > c.Audio.MaxBufferedFrames {
		errors = append(errors, fmt.Sprintf("buffered frames must be between %d and %d", c.Audio.MinBufferedFrames, c.Audio.MaxBufferedFrames))
	}

	// Validate queue configuration
	if c.Queue.MaxSize <= 0 {
		errors = append(errors, "max queue size must be greater than 0")
	}

	if c.Queue.MaxPlaylistSize <= 0 {
		errors = append(errors, "max playlist size must be greater than 0")
	}

	if c.Queue.MaxConcurrentPlaylists <= 0 {
		errors = append(errors, "max concurrent playlists must be greater than 0")
	}

	// Validate cache configuration
	if c.Cache.MaxCacheSize <= 0 {
		errors = append(errors, "max cache size must be greater than 0")
	}

	if c.Cache.BufferSize <= 0 {
		errors = append(errors, "buffer size must be greater than 0")
	}

	// Validate logging configuration
	validLogLevels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
	if !contains(validLogLevels, c.Logging.Level) {
		errors = append(errors, fmt.Sprintf("log level must be one of: %s", strings.Join(validLogLevels, ", ")))
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// contains checks if a string slice contains a value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetDCAOptions returns DCA encoder options based on configuration
func (c *Config) GetDCAOptions() map[string]interface{} {
	return map[string]interface{}{
		"bitrate":            c.Audio.Bitrate,
		"volume":             c.Audio.Volume,
		"frame_rate":         c.Audio.FrameRate,
		"frame_duration":     c.Audio.FrameDuration,
		"compression_level":  c.Audio.CompressionLevel,
		"packet_loss":        c.Audio.PacketLoss,
		"buffered_frames":    c.Audio.BufferedFrames,
		"enable_vbr":         c.Audio.EnableVBR,
	}
}

// GetMemoryEstimate returns an estimate of memory usage in bytes
func (c *Config) GetMemoryEstimate() int64 {
	// Rough estimation based on audio buffer configuration
	bytesPerFrame := int64(c.Audio.FrameRate * 2 * 2) // 16-bit stereo
	bufferMemory := int64(c.Audio.BufferedFrames) * bytesPerFrame
	
	// Add cache memory estimate
	cacheMemory := c.Cache.MaxCacheSize / 10 // Assume 10% of cache is kept in memory
	
	return bufferMemory + cacheMemory
}

// IsExperimentalFeatureEnabled checks if an experimental feature is enabled
func (c *Config) IsExperimentalFeatureEnabled(feature string) bool {
	for _, f := range c.Features.ExperimentalFeatures {
		if f == feature {
			return true
		}
	}
	return false
}

// GetRedactedToken returns a redacted version of the token for logging
func (c *Config) GetRedactedToken() string {
	if len(c.Discord.Token) < 8 {
		return "***"
	}
	return c.Discord.Token[:8] + "***"
}

// GetRedactedAPIKey returns a redacted version of the API key for logging
func (c *Config) GetRedactedAPIKey() string {
	if len(c.YouTube.APIKey) < 8 {
		return "***"
	}
	return c.YouTube.APIKey[:8] + "***"
}