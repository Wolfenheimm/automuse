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

// Sets up the DCA encoder options
func init() {
	// Set up DCA options exactly like original implementation
	opts = dca.StdEncodeOptions
	opts.RawOutput = false
	opts.Bitrate = 128
	opts.Application = dca.AudioApplicationLowDelay
	opts.Volume = 256 // Increasing volume for better audibility
	opts.CompressionLevel = 10
	opts.FrameRate = 48000
	opts.FrameDuration = 20
	opts.PacketLoss = 1
	opts.VBR = true
	opts.BufferedFrames = 17000

	// Log DCA options for debugging
	log.Println("DCA options initialized:")
	log.Printf("- Bitrate: %d, Volume: %d", opts.Bitrate, opts.Volume)
	log.Printf("- Application: %s, FrameRate: %d", opts.Application, opts.FrameRate)
	log.Printf("- BufferedFrames: %d, VBR: %t", opts.BufferedFrames, opts.VBR)
}
