package main

import (
	"context"
	"sync"

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
	stopRequested   bool // Flag to prevent queue processing after stop command
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
	bufferManager   = NewBufferManager(5) // Increased from 3 to 5 songs for better skip performance
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
