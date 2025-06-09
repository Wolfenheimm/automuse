package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SongMetadata represents the metadata for a cached song
type SongMetadata struct {
	VideoID      string    `json:"video_id"`
	Title        string    `json:"title"`
	Duration     string    `json:"duration"`
	FilePath     string    `json:"file_path"`
	FileSize     int64     `json:"file_size"`
	DownloadedAt time.Time `json:"downloaded_at"`
	LastUsed     time.Time `json:"last_used"`
	UseCount     int       `json:"use_count"`
	Artist       string    `json:"artist,omitempty"`
	Album        string    `json:"album,omitempty"`
	TitleHash    string    `json:"title_hash"` // For similarity matching
}

// MetadataManager handles song metadata operations
type MetadataManager struct {
	filePath string
	metadata map[string]*SongMetadata
	mutex    sync.RWMutex
}

// NewMetadataManager creates a new metadata manager
func NewMetadataManager(filePath string) *MetadataManager {
	mm := &MetadataManager{
		filePath: filePath,
		metadata: make(map[string]*SongMetadata),
	}
	mm.loadMetadata()
	return mm
}

// loadMetadata loads existing metadata from JSON file
func (mm *MetadataManager) loadMetadata() error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	// Create metadata file if it doesn't exist
	if _, err := os.Stat(mm.filePath); os.IsNotExist(err) {
		log.Printf("INFO: Creating new metadata file: %s", mm.filePath)
		return mm.saveMetadataUnsafe()
	}

	data, err := os.ReadFile(mm.filePath)
	if err != nil {
		log.Printf("ERROR: Failed to read metadata file: %v", err)
		return err
	}

	if len(data) == 0 {
		log.Printf("INFO: Empty metadata file, initializing with empty data")
		return nil
	}

	err = json.Unmarshal(data, &mm.metadata)
	if err != nil {
		log.Printf("ERROR: Failed to parse metadata JSON: %v", err)
		return err
	}

	log.Printf("INFO: Loaded metadata for %d songs", len(mm.metadata))
	return nil
}

// saveMetadata saves current metadata to JSON file
func (mm *MetadataManager) saveMetadata() error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	return mm.saveMetadataUnsafe()
}

// saveMetadataUnsafe saves metadata without locking (internal use)
func (mm *MetadataManager) saveMetadataUnsafe() error {
	data, err := json.MarshalIndent(mm.metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(mm.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %v", err)
	}

	err = os.WriteFile(mm.filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write metadata file: %v", err)
	}

	return nil
}

// AddSong adds or updates song metadata
func (mm *MetadataManager) AddSong(videoID, title, duration, filePath string, fileSize int64) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	// Extract artist from title if possible
	artist := extractArtistFromTitle(title)
	titleHash := generateTitleHash(title)

	metadata := &SongMetadata{
		VideoID:      videoID,
		Title:        title,
		Duration:     duration,
		FilePath:     filePath,
		FileSize:     fileSize,
		DownloadedAt: time.Now(),
		LastUsed:     time.Now(),
		UseCount:     1,
		Artist:       artist,
		TitleHash:    titleHash,
	}

	// Update existing or add new
	if existing, exists := mm.metadata[videoID]; exists {
		metadata.DownloadedAt = existing.DownloadedAt
		metadata.UseCount = existing.UseCount + 1
	}

	mm.metadata[videoID] = metadata
	log.Printf("INFO: Added/updated metadata for: %s (%s)", title, videoID)

	return mm.saveMetadataUnsafe()
}

// GetSong retrieves song metadata by video ID
func (mm *MetadataManager) GetSong(videoID string) (*SongMetadata, bool) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	if metadata, exists := mm.metadata[videoID]; exists {
		// Update last used time
		metadata.LastUsed = time.Now()
		return metadata, true
	}

	return nil, false
}

// HasSong checks if a song exists in cache
func (mm *MetadataManager) HasSong(videoID string) bool {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	metadata, exists := mm.metadata[videoID]
	if !exists {
		return false
	}

	// Verify file still exists
	if _, err := os.Stat(metadata.FilePath); os.IsNotExist(err) {
		log.Printf("WARN: Cached file missing, removing from metadata: %s", metadata.FilePath)
		// Remove from metadata in a separate goroutine to avoid deadlock
		go mm.RemoveSong(videoID)
		return false
	}

	return true
}

// FindSimilarSongs finds songs with similar titles to detect potential duplicates
func (mm *MetadataManager) FindSimilarSongs(title string, threshold float64) []*SongMetadata {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	var similar []*SongMetadata
	targetHash := generateTitleHash(title)

	for _, metadata := range mm.metadata {
		similarity := calculateSimilarity(targetHash, metadata.TitleHash)
		if similarity >= threshold {
			similar = append(similar, metadata)
		}
	}

	return similar
}

// FindByTitle searches for songs by exact or partial title match
func (mm *MetadataManager) FindByTitle(searchTitle string) []*SongMetadata {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	var matches []*SongMetadata
	searchLower := strings.ToLower(searchTitle)

	for _, metadata := range mm.metadata {
		titleLower := strings.ToLower(metadata.Title)
		if strings.Contains(titleLower, searchLower) {
			matches = append(matches, metadata)
		}
	}

	return matches
}

// RemoveSong removes song metadata
func (mm *MetadataManager) RemoveSong(videoID string) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if _, exists := mm.metadata[videoID]; exists {
		delete(mm.metadata, videoID)
		log.Printf("INFO: Removed metadata for video ID: %s", videoID)
		return mm.saveMetadataUnsafe()
	}

	return nil
}

// CleanupMissing removes metadata entries for files that no longer exist
func (mm *MetadataManager) CleanupMissing() error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	var toRemove []string
	for videoID, metadata := range mm.metadata {
		if _, err := os.Stat(metadata.FilePath); os.IsNotExist(err) {
			toRemove = append(toRemove, videoID)
		}
	}

	if len(toRemove) > 0 {
		for _, videoID := range toRemove {
			delete(mm.metadata, videoID)
			log.Printf("INFO: Cleaned up missing file metadata: %s", videoID)
		}
		return mm.saveMetadataUnsafe()
	}

	return nil
}

// GetStats returns cache statistics
func (mm *MetadataManager) GetStats() map[string]interface{} {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	var totalSize int64
	oldestDownload := time.Now()
	mostUsed := 0

	for _, metadata := range mm.metadata {
		totalSize += metadata.FileSize
		if metadata.DownloadedAt.Before(oldestDownload) {
			oldestDownload = metadata.DownloadedAt
		}
		if metadata.UseCount > mostUsed {
			mostUsed = metadata.UseCount
		}
	}

	return map[string]interface{}{
		"total_songs":     len(mm.metadata),
		"total_size_mb":   totalSize / (1024 * 1024),
		"oldest_download": oldestDownload,
		"highest_use":     mostUsed,
	}
}

// StatsData represents detailed cache statistics
type StatsData struct {
	TotalSongs   int
	TotalPlays   int
	AverageUsage float64
	TopSongs     []*SongMetadata
}

// GetDetailedStats returns detailed cache statistics including top songs
func (mm *MetadataManager) GetDetailedStats() StatsData {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	var totalPlays int
	var songs []*SongMetadata

	for _, metadata := range mm.metadata {
		totalPlays += metadata.UseCount
		songs = append(songs, metadata)
	}

	// Sort songs by use count (descending)
	for i := 0; i < len(songs)-1; i++ {
		for j := i + 1; j < len(songs); j++ {
			if songs[j].UseCount > songs[i].UseCount {
				songs[i], songs[j] = songs[j], songs[i]
			}
		}
	}

	averageUsage := 0.0
	if len(songs) > 0 {
		averageUsage = float64(totalPlays) / float64(len(songs))
	}

	return StatsData{
		TotalSongs:   len(songs),
		TotalPlays:   totalPlays,
		AverageUsage: averageUsage,
		TopSongs:     songs,
	}
}

// GetOldSongs returns songs older than the specified duration
func (mm *MetadataManager) GetOldSongs(maxAge time.Duration) []*SongMetadata {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	cutoff := time.Now().Add(-maxAge)
	var oldSongs []*SongMetadata

	for _, metadata := range mm.metadata {
		// Consider a song old if both downloaded and last used are before cutoff
		if metadata.DownloadedAt.Before(cutoff) && metadata.LastUsed.Before(cutoff) {
			oldSongs = append(oldSongs, metadata)
		}
	}

	return oldSongs
}

// SaveMetadata exposes the save functionality publicly
func (mm *MetadataManager) SaveMetadata() error {
	return mm.saveMetadata()
}

// extractArtistFromTitle attempts to extract artist name from title
func extractArtistFromTitle(title string) string {
	// Common separators between artist and song
	separators := []string{" - ", " – ", " | ", " || ", ": "}

	for _, sep := range separators {
		if parts := strings.Split(title, sep); len(parts) >= 2 {
			artist := strings.TrimSpace(parts[0])
			// Don't return very short artists (likely false positives)
			if len(artist) > 2 && len(artist) < 50 {
				return artist
			}
		}
	}

	return ""
}

// generateTitleHash creates a simplified hash of the title for similarity comparison
func generateTitleHash(title string) string {
	// Normalize title for comparison
	normalized := strings.ToLower(title)

	// Remove common noise words and characters
	noise := []string{
		"official", "video", "music", "hd", "hq", "lyric", "lyrics",
		"audio", "version", "remix", "remaster", "feat", "ft",
		"(", ")", "[", "]", "|", "-", "_", ".", ",",
	}

	for _, n := range noise {
		normalized = strings.ReplaceAll(normalized, n, " ")
	}

	// Split into words and sort for consistent hashing
	words := strings.Fields(normalized)
	if len(words) == 0 {
		return title
	}

	// Take first few significant words
	maxWords := 5
	if len(words) < maxWords {
		maxWords = len(words)
	}

	return strings.Join(words[:maxWords], " ")
}

// calculateSimilarity calculates similarity between two title hashes
func calculateSimilarity(hash1, hash2 string) float64 {
	if hash1 == hash2 {
		return 1.0
	}

	words1 := strings.Fields(hash1)
	words2 := strings.Fields(hash2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Simple word overlap similarity
	common := 0
	for _, word1 := range words1 {
		for _, word2 := range words2 {
			if word1 == word2 && len(word1) > 2 { // Only count significant words
				common++
				break
			}
		}
	}

	// Jaccard similarity
	union := len(words1) + len(words2) - common
	if union == 0 {
		return 0.0
	}

	return float64(common) / float64(union)
}
