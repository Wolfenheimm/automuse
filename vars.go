package main

import (
	"context"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	yt "github.com/kkdai/youtube/v2"
	"google.golang.org/api/youtube/v3"
)

// Bot Parameters
var (
	searchRequested bool
	stopRequested   bool         // Flag to prevent queue processing after stop command
	stopMutex       sync.RWMutex // Mutex for thread-safe stopRequested access
	playbackEnding  bool         // Flag to indicate playback is ending naturally
	playbackMutex   sync.RWMutex // Mutex for thread-safe playbackEnding access

	// Rate limiting and resource management
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
	client          = yt.Client{}   // Enable debug mode
	ctx             context.Context // Assigned from main application context
	song            = Song{}
	searchQueue     = []SongSearch{}
	queue           = []Song{}
	queueMutex      sync.Mutex       // Mutex for thread-safe queue operations
	metadataManager *MetadataManager // Metadata manager for song caching
	bufferManager   *BufferManager   // Pre-download buffer manager (initialized conditionally)

	// Error handling and command system
	errorHandler    *ErrorHandler       // Global error handler
	commandHandlers = []CommandHandler{ // Command handler system
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

// Initialize rate limiting resources
func init() {
	playlistSemaphore = make(chan struct{}, maxConcurrentPlaylists)
	activeCommands = make(map[string]time.Time)
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
