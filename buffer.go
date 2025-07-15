package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// BufferManager handles pre-downloading songs to maintain a buffer
type BufferManager struct {
	downloadQueue []Song
	downloading   map[string]bool   // Track which songs are currently downloading
	failedVideos  map[string]int    // Track failed download attempts per video ID
	lastFailTime  map[string]time.Time // Track when each video last failed
	mutex         sync.RWMutex
	maxBuffer     int
	session       *discordgo.Session
	channelID     string
	isActive      bool
	stopChan      chan struct{} // Channel to signal goroutine shutdown
}

// NewBufferManager creates a new buffer manager
func NewBufferManager(maxBuffer int) *BufferManager {
	return &BufferManager{
		downloadQueue: make([]Song, 0),
		downloading:   make(map[string]bool),
		failedVideos:  make(map[string]int),
		lastFailTime:  make(map[string]time.Time),
		maxBuffer:     maxBuffer,
		isActive:      false,
		stopChan:      make(chan struct{}),
	}
}

// StartBuffering begins the buffering process for the current queue
func (bm *BufferManager) StartBuffering(session *discordgo.Session, channelID string) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.session = session
	bm.channelID = channelID
	bm.isActive = true

	log.Printf("INFO: Starting buffer manager with max buffer size: %d", bm.maxBuffer)

	// Start the background buffer maintenance goroutine
	go bm.maintainBuffer()
}

// StopBuffering stops the buffering process
func (bm *BufferManager) StopBuffering() {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	if bm.isActive {
		bm.isActive = false

		// Signal shutdown to maintenance goroutine
		select {
		case bm.stopChan <- struct{}{}:
		default:
			// Channel might be full or closed, that's okay
		}
	}

	bm.downloadQueue = make([]Song, 0)
	bm.downloading = make(map[string]bool)
	bm.failedVideos = make(map[string]int)
	bm.lastFailTime = make(map[string]time.Time)

	log.Printf("INFO: Stopped buffer manager")
}

// UpdateQueue updates the buffer manager with the current queue state
func (bm *BufferManager) UpdateQueue(currentQueue []Song, currentPlayingIndex int) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	if !bm.isActive {
		return
	}

	// Calculate which songs need to be in the buffer
	var songsToBuffer []Song
	startIndex := currentPlayingIndex + 1 // Start with next song after currently playing

	for i := 0; i < bm.maxBuffer && startIndex+i < len(currentQueue); i++ {
		song := currentQueue[startIndex+i]
		songsToBuffer = append(songsToBuffer, song)
	}

	// Update the download queue
	bm.downloadQueue = songsToBuffer

	log.Printf("INFO: Updated buffer queue with %d songs to buffer", len(songsToBuffer))
}

// shouldSkipDownload checks if a video should be skipped due to previous failures
func (bm *BufferManager) shouldSkipDownload(videoID string) bool {
	const maxRetries = 3
	const backoffDuration = 30 * time.Minute // Wait 30 minutes before retrying failed videos
	
	failures, hasFailed := bm.failedVideos[videoID]
	if !hasFailed {
		return false // Never failed, proceed with download
	}
	
	// If failed too many times, permanently skip
	if failures >= maxRetries {
		return true
	}
	
	// If failed recently, wait for backoff period
	lastFail, hasTime := bm.lastFailTime[videoID]
	if hasTime && time.Since(lastFail) < backoffDuration {
		return true
	}
	
	return false
}

// recordFailure records a download failure for a video
func (bm *BufferManager) recordFailure(videoID string) {
	bm.failedVideos[videoID]++
	bm.lastFailTime[videoID] = time.Now()
	
	failures := bm.failedVideos[videoID]
	if failures >= 3 {
		log.Printf("WARN: Video %s permanently failed after %d attempts, will not retry", videoID, failures)
	} else {
		log.Printf("WARN: Video %s failed (attempt %d/3), will retry after 30 minutes", videoID, failures)
	}
}

// PreDownloadInitialSongs downloads the first few songs before starting playback
func (bm *BufferManager) PreDownloadInitialSongs(songs []Song, session *discordgo.Session, channelID string) error {
	if len(songs) == 0 {
		return nil
	}

	bm.session = session
	bm.channelID = channelID

	// Determine how many songs to pre-download (up to maxBuffer or all songs if fewer)
	downloadCount := bm.maxBuffer
	if len(songs) < downloadCount {
		downloadCount = len(songs)
	}

	songsToDownload := songs[:downloadCount]

	log.Printf("INFO: Pre-downloading %d songs before starting playback", downloadCount)

	if len(songsToDownload) > 1 {
		session.ChannelMessageSend(channelID, fmt.Sprintf("**[Muse]** Pre-downloading %d songs for smooth playback... :hourglass:", downloadCount))
	}

	// Download songs in parallel with limited concurrency
	maxConcurrent := 4 // Increased from 2 to 4 for faster downloads
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	downloadResults := make(chan bool, len(songsToDownload))

	for i, song := range songsToDownload {
		wg.Add(1)
		go func(song Song, index int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			success := bm.downloadSong(song, index+1, downloadCount)
			downloadResults <- success
		}(song, i)
	}

	// Wait for all downloads to complete
	wg.Wait()
	close(downloadResults)

	// Check results
	successCount := 0
	for success := range downloadResults {
		if success {
			successCount++
		}
	}

	if successCount > 0 {
		if successCount == downloadCount {
			session.ChannelMessageSend(channelID, fmt.Sprintf("**[Muse]** All %d songs pre-downloaded! Starting playback... :musical_note:", successCount))
		} else {
			session.ChannelMessageSend(channelID, fmt.Sprintf("**[Muse]** Pre-downloaded %d/%d songs. Starting playback... :musical_note:", successCount, downloadCount))
		}
	}

	log.Printf("INFO: Pre-download completed. Success: %d/%d", successCount, downloadCount)
	return nil
}

// maintainBuffer runs in background to maintain the download buffer
func (bm *BufferManager) maintainBuffer() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bm.stopChan:
			log.Printf("INFO: Buffer maintenance goroutine shutting down")
			return
		case <-ticker.C:
			bm.mutex.RLock()
			if !bm.isActive {
				bm.mutex.RUnlock()
				return
			}

			// Check which songs in the buffer need downloading
			var songsToDownload []Song
			for _, song := range bm.downloadQueue {
				// Skip if already cached
				if metadataManager.HasSong(song.VidID) {
					continue
				}

				// Skip if currently downloading
				if bm.downloading[song.VidID] {
					continue
				}

				// Skip if failed too many times or in backoff period
				if bm.shouldSkipDownload(song.VidID) {
					continue
				}

				songsToDownload = append(songsToDownload, song)
			}
			bm.mutex.RUnlock()

			// Download songs that need downloading (limit concurrent downloads)
			if len(songsToDownload) > 0 {
				for _, song := range songsToDownload[:min(4, len(songsToDownload))] {
					go func(s Song) {
						bm.mutex.Lock()
						bm.downloading[s.VidID] = true
						bm.mutex.Unlock()

						success := bm.downloadSong(s, 0, 0) // 0 index means background download

						bm.mutex.Lock()
						delete(bm.downloading, s.VidID)
						if !success {
							// Record the failure for this video
							bm.recordFailure(s.VidID)
						}
						bm.mutex.Unlock()

						if success {
							log.Printf("INFO: Background buffer download completed: %s", s.Title)
						}
					}(song)
				}
			}
		}
	}
}

// downloadSong downloads a single song and updates metadata
func (bm *BufferManager) downloadSong(song Song, progressIndex, totalCount int) bool {
	// Check if already cached
	if metadataManager.HasSong(song.VidID) {
		if progressIndex > 0 {
			log.Printf("INFO: Song %d/%d already cached: %s", progressIndex, totalCount, song.Title)
		}
		return true
	}

	if progressIndex > 0 {
		log.Printf("INFO: Downloading song %d/%d: %s", progressIndex, totalCount, song.Title)
	} else {
		log.Printf("INFO: Background downloading: %s", song.Title)
	}

	// Use the existing downloadSongToCache function
	success := downloadSongToCache(song)

	if success && progressIndex > 0 && bm.session != nil && bm.channelID != "" {
		// Show progress for initial downloads only
		if progressIndex == totalCount {
			bm.session.ChannelMessageSend(bm.channelID, fmt.Sprintf("**[Muse]** Downloaded %d/%d songs :white_check_mark:", progressIndex, totalCount))
		}
	}

	return success
}

// downloadSongToCache downloads a song and saves it to cache with metadata
func downloadSongToCache(song Song) bool {
	log.Printf("INFO: Downloading to cache: %s (ID: %s)", song.Title, song.VidID)

	// For manual/local files, they're already "downloaded"
	if song.VideoURL == "" || song.VideoURL == song.Title {
		log.Printf("INFO: Local file, marking as cached: %s", song.Title)
		return true
	}

	// Check if it's already a file path
	if strings.HasPrefix(song.VideoURL, "downloads/") {
		log.Printf("INFO: Already downloaded file: %s", song.VideoURL)
		return true
	}

	// For YouTube URLs, download using yt-dlp
	// Always use the YouTube URL format for yt-dlp, not the stream URL
	videoID := song.VidID
	originalURL := "https://www.youtube.com/watch?v=" + videoID

	// Create downloads directory if it doesn't exist
	downloadDir := "downloads"
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		log.Printf("ERROR: Failed to create downloads directory: %v", err)
		return false
	}

	// Define MP3 path
	mp3Path := filepath.Join(downloadDir, videoID+".mp3")

	// Check if already exists
	if _, err := os.Stat(mp3Path); err == nil {
		log.Printf("INFO: File already exists, adding to metadata: %s", mp3Path)
		if fileInfo, statErr := os.Stat(mp3Path); statErr == nil {
			metadataManager.AddSong(videoID, song.Title, song.Duration, mp3Path, fileInfo.Size())
		}
		return true
	}

	// Download using yt-dlp
	env := os.Environ()
	env = append(env, "YT_TOKEN="+os.Getenv("YT_TOKEN"))

	cmd := exec.Command("yt-dlp",
		"--no-playlist",         // Don't download playlists
		"-x",                    // Extract audio
		"--audio-format", "mp3", // Convert to MP3
		"--audio-quality", "256K", // Increased from 192K to 256K for better quality
		"--no-warnings", // Reduce noise in logs
		"--progress",    // Show download progress for buffer downloads
		"-o", mp3Path,   // Output file
		originalURL) // Original YouTube URL

	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ERROR: Download failed for %s: %v", song.Title, err)
		log.Printf("yt-dlp output: %s", string(output))
		// Clean up partial file if it exists
		os.Remove(mp3Path)
		return false
	}

	// Verify the MP3 file exists and has content
	if fileInfo, err := os.Stat(mp3Path); err != nil || fileInfo.Size() == 0 {
		log.Printf("ERROR: Downloaded file is missing or empty: %s", mp3Path)
		if err == nil {
			os.Remove(mp3Path)
		}
		return false
	} else {
		log.Printf("INFO: Successfully downloaded: %s (size: %d bytes)", mp3Path, fileInfo.Size())

		// Add to metadata manager
		if err := metadataManager.AddSong(videoID, song.Title, song.Duration, mp3Path, fileInfo.Size()); err != nil {
			log.Printf("WARN: Failed to add downloaded song to metadata: %v", err)
		}
	}

	return true
}
