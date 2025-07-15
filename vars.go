package main

import (
	"context"
	"sync"
	"time"

	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	yt "github.com/kkdai/youtube/v2"
	"google.golang.org/api/youtube/v3"
)

// Bot Parameters
var (
	botToken        string
	youtubeToken    string
	searchRequested bool
	stopRequested   bool         // Flag to prevent queue processing after stop command
	stopMutex       sync.RWMutex // Mutex for thread-safe stopRequested access
	playbackEnding  bool         // Flag to indicate playback is ending naturally
	playbackMutex   sync.RWMutex // Mutex for thread-safe playbackEnding access

	// Rate limiting and resource management
	playlistProcessing     bool                         // Flag to indicate playlist processing is active
	playlistMutex          sync.RWMutex                 // Mutex for playlist processing flag
	lastPlaylistTime       time.Time                    // Last time a playlist was processed
	playlistCooldown       = 5 * time.Second            // Minimum time between playlist commands
	maxConcurrentPlaylists = 3                          // Maximum number of playlists that can be processed simultaneously
	playlistSemaphore      chan struct{}                // Semaphore to limit concurrent playlist processing
	userRateLimit          = make(map[string]time.Time) // Per-user rate limiting
	userRateMutex          sync.RWMutex                 // Mutex for user rate limiting
	maxQueueSize           = 500                        // Maximum total queue size to prevent memory issues

	// Playback state protection - prevent multiple simultaneous playback
	isPlaying          bool
	playbackStateMutex sync.RWMutex

	// Command deduplication to prevent duplicate command processing
	activeCommands map[string]time.Time // Track active commands by user+command
	commandMutex   sync.RWMutex         // Mutex for command tracking

	service         *youtube.Service
	s               *discordgo.Session
	v               = new(VoiceInstance)
	opts            = dca.StdEncodeOptions
	client          = yt.Client{} // Enable debug mode
	ctx             = context.Background()
	song            = Song{}
	searchQueue     = []SongSearch{}
	queue           = []Song{}
	queueMutex      sync.Mutex            // Mutex for thread-safe queue operations
	metadataManager *MetadataManager      // Metadata manager for song caching
	bufferManager   *BufferManager        // Pre-download buffer manager (initialized conditionally)
	
	// Error handling and command system
	errorHandler    *ErrorHandler         // Global error handler
	commandHandlers = []CommandHandler{   // Command handler system
		&PlayHelpCommand{},
		&PlayStuffCommand{},
		&PlayKudasaiCommand{},
		&PlayCommand{},
		&StopCommand{},
		&SkipCommand{},
		&PauseCommand{},
		&ResumeCommand{},
		&QueueCommand{},
		&RemoveCommand{},
		&CacheCommand{},
		&CacheClearCommand{},
		&BufferStatusCommand{},
		&MoveQueueCommand{},
		&ShuffleQueueCommand{},
		&EmergencyResetCommand{},
	}
)

// Configuration constants for better maintainability
const (
	// Audio quality settings
	DefaultBitrate       = 128   // 128kbps - good balance of quality and bandwidth
	DefaultVolume        = 256   // Discord volume level
	DefaultFrameRate     = 48000 // 48kHz sample rate
	DefaultFrameDuration = 20    // 20ms frame duration (standard)

	// Buffer settings - CRITICAL: Keep these values reasonable!
	// Each frame = ~20ms of audio, so 200 frames = ~4 seconds buffer
	SafeBufferedFrames = 200 // 4 seconds of audio buffer (was 17000 = 5.7 minutes!)
	MaxBufferedFrames  = 500 // Maximum safe buffer size
	MinBufferedFrames  = 100 // Minimum for stability

	// Performance settings
	DefaultCompressionLevel = 5 // Balanced compression (0-10 scale)
	DefaultPacketLoss       = 1 // Packet loss compensation
)

// Initialize rate limiting resources
func init() {
	playlistSemaphore = make(chan struct{}, maxConcurrentPlaylists)
	activeCommands = make(map[string]time.Time)
}

// Sets up the DCA encoder options with safe parameters
func init() {
	// Set up DCA options with safe, performance-optimized values
	opts = dca.StdEncodeOptions
	opts.RawOutput = false
	opts.Bitrate = DefaultBitrate
	opts.Application = dca.AudioApplicationAudio // Changed from LowDelay for better quality
	opts.Volume = DefaultVolume
	opts.CompressionLevel = DefaultCompressionLevel // Reduced from 10 for better performance
	opts.FrameRate = DefaultFrameRate
	opts.FrameDuration = DefaultFrameDuration
	opts.PacketLoss = DefaultPacketLoss
	opts.VBR = true
	opts.BufferedFrames = SafeBufferedFrames // FIXED: Was 17000 (unsafe), now 200 (safe)

	// Validate buffer settings for safety
	if opts.BufferedFrames > MaxBufferedFrames {
		log.Printf("WARNING: BufferedFrames (%d) exceeds safe maximum (%d), capping to safe value",
			opts.BufferedFrames, MaxBufferedFrames)
		opts.BufferedFrames = MaxBufferedFrames
	}

	if opts.BufferedFrames < MinBufferedFrames {
		log.Printf("WARNING: BufferedFrames (%d) below minimum (%d), setting to minimum",
			opts.BufferedFrames, MinBufferedFrames)
		opts.BufferedFrames = MinBufferedFrames
	}

	// Log DCA options for debugging and verification
	log.Println("DCA options initialized with safe parameters:")
	log.Printf("- Bitrate: %dkbps, Volume: %d", opts.Bitrate, opts.Volume)
	log.Printf("- Application: %s, FrameRate: %dHz", opts.Application, opts.FrameRate)
	log.Printf("- BufferedFrames: %d (~%.1fs buffer), VBR: %t",
		opts.BufferedFrames, float64(opts.BufferedFrames*opts.FrameDuration)/1000, opts.VBR)
	log.Printf("- CompressionLevel: %d, PacketLoss: %d", opts.CompressionLevel, opts.PacketLoss)

	// Calculate estimated memory usage for the buffer
	estimatedMemoryMB := float64(opts.BufferedFrames*opts.FrameRate*2*2) / (1024 * 1024) // Rough estimate
	log.Printf("- Estimated buffer memory usage: ~%.1fMB", estimatedMemoryMB)
}

// Thread-safe functions for stopRequested flag
func setStopRequested(value bool) {
	stopMutex.Lock()
	defer stopMutex.Unlock()
	stopRequested = value
}

func isStopRequested() bool {
	stopMutex.RLock()
	defer stopMutex.RUnlock()
	return stopRequested
}

// Thread-safe functions for playbackEnding flag
func setPlaybackEnding(value bool) {
	playbackMutex.Lock()
	defer playbackMutex.Unlock()
	playbackEnding = value
}

func isPlaybackEnding() bool {
	playbackMutex.RLock()
	defer playbackMutex.RUnlock()
	return playbackEnding
}

// Thread-safe functions for playlist processing management
func setPlaylistProcessing(value bool) {
	playlistMutex.Lock()
	defer playlistMutex.Unlock()
	playlistProcessing = value
	if value {
		lastPlaylistTime = time.Now()
	}
}

func isPlaylistProcessing() bool {
	playlistMutex.RLock()
	defer playlistMutex.RUnlock()
	return playlistProcessing
}

func canProcessPlaylist() bool {
	playlistMutex.RLock()
	defer playlistMutex.RUnlock()
	return time.Since(lastPlaylistTime) >= playlistCooldown
}

// Per-user rate limiting
func checkUserRateLimit(userID string) bool {
	userRateMutex.Lock()
	defer userRateMutex.Unlock()

	lastTime, exists := userRateLimit[userID]
	if !exists || time.Since(lastTime) >= 3*time.Second {
		userRateLimit[userID] = time.Now()
		return true
	}
	return false
}

// Thread-safe functions for playback state management
func setPlaybackState(playing bool) {
	playbackStateMutex.Lock()
	defer playbackStateMutex.Unlock()
	isPlaying = playing
}

func getPlaybackState() bool {
	playbackStateMutex.RLock()
	defer playbackStateMutex.RUnlock()
	return isPlaying
}

// Command deduplication to prevent duplicate processing
func isCommandActive(userID, command string) bool {
	commandMutex.RLock()
	defer commandMutex.RUnlock()

	key := userID + ":" + command
	lastTime, exists := activeCommands[key]
	if !exists {
		return false
	}

	// Playlist commands have longer timeout due to processing time
	if command == "playlist" {
		return time.Since(lastTime) < 10*time.Second
	}

	// Regular commands have shorter timeout
	return time.Since(lastTime) < 2*time.Second
}

func setCommandActive(userID, command string) {
	commandMutex.Lock()
	defer commandMutex.Unlock()

	key := userID + ":" + command
	activeCommands[key] = time.Now()
}

func clearCommandActive(userID, command string) {
	commandMutex.Lock()
	defer commandMutex.Unlock()

	key := userID + ":" + command
	delete(activeCommands, key)
}

// Atomic playlist processing protection - returns true if successfully acquired, false if already processing
func atomicSetPlaylistProcessing(value bool) bool {
	playlistMutex.Lock()
	defer playlistMutex.Unlock()

	if value && playlistProcessing {
		// Someone else is already processing
		return false
	}

	playlistProcessing = value
	if value {
		lastPlaylistTime = time.Now()
	}
	return true
}
