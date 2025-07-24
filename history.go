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

// HistoryEntry represents a single played song in the history
type HistoryEntry struct {
	Song      Song      `json:"song"`       // The song that was played
	PlayedAt  time.Time `json:"played_at"`  // When it was played
	GuildID   string    `json:"guild_id"`   // Which Discord server
	GuildName string    `json:"guild_name"` // Server name for easier identification
	Duration  int64     `json:"duration"`   // How long it played (in seconds)
}

// GuildHistory represents the history for a specific guild
type GuildHistory struct {
	GuildID     string         `json:"guild_id"`
	GuildName   string         `json:"guild_name"`
	Entries     []HistoryEntry `json:"entries"`
	LastUpdated time.Time      `json:"last_updated"`
}

// HistoryManager manages song history across all guilds
type HistoryManager struct {
	histories    map[string]*GuildHistory // guild_id -> history
	mutex        sync.RWMutex            // Protect concurrent access
	maxEntries   int                     // Maximum entries per guild
	dataFile     string                  // File to persist history data
	autosave     bool                    // Whether to autosave after changes
}

// HistoryConfig holds configuration for the history system
type HistoryConfig struct {
	MaxEntries       int    // Maximum history entries per guild (default: 50)
	DataFile         string // File to store history data
	EnableAutosave   bool   // Auto-save after each addition
	EnablePersistence bool  // Whether to persist history to disk
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(config HistoryConfig) *HistoryManager {
	if config.MaxEntries <= 0 {
		config.MaxEntries = 50 // Default to 50 entries
	}
	if config.DataFile == "" {
		config.DataFile = "downloads/history.json" // Default location
	}

	hm := &HistoryManager{
		histories:  make(map[string]*GuildHistory),
		maxEntries: config.MaxEntries,
		dataFile:   config.DataFile,
		autosave:   config.EnableAutosave,
	}

	// Load existing history if persistence is enabled
	if config.EnablePersistence {
		if err := hm.LoadHistory(); err != nil {
			log.Printf("WARN: Failed to load history data: %v", err)
		}
	}

	return hm
}

// AddEntry adds a new song to the history for a specific guild
func (hm *HistoryManager) AddEntry(song Song, guildID, guildName string, duration time.Duration) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	// Get or create guild history  
	history, exists := hm.histories[guildID]
	if !exists {
		history = &GuildHistory{
			GuildID:   guildID,
			GuildName: guildName,
			Entries:   make([]HistoryEntry, 0, hm.maxEntries),
		}
		hm.histories[guildID] = history
	}

	// Update guild name in case it changed
	history.GuildName = guildName

	// Create new history entry
	entry := HistoryEntry{
		Song:      song,
		PlayedAt:  time.Now(),
		GuildID:   guildID,
		GuildName: guildName,
		Duration:  int64(duration.Seconds()),
	}

	// Add entry to the beginning (most recent first)
	history.Entries = append([]HistoryEntry{entry}, history.Entries...)

	// Trim to max entries if needed
	if len(history.Entries) > hm.maxEntries {
		history.Entries = history.Entries[:hm.maxEntries]
	}

	// Update timestamp
	history.LastUpdated = time.Now()

	// Auto-save if enabled
	if hm.autosave {
		go func() {
			if err := hm.SaveHistory(); err != nil {
				log.Printf("WARN: Failed to auto-save history: %v", err)
			}
		}()
	}

	log.Printf("INFO: Added song to history for guild %s: %s", guildID, song.Title)
	return nil
}

// GetHistory returns the history for a specific guild
func (hm *HistoryManager) GetHistory(guildID string, limit int) ([]HistoryEntry, error) {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	history, exists := hm.histories[guildID]
	if !exists {
		return []HistoryEntry{}, nil // Empty history
	}

	// Apply limit if specified
	entries := history.Entries
	if limit > 0 && limit < len(entries) {
		entries = entries[:limit]
	}

	// Return a copy to prevent external modification
	result := make([]HistoryEntry, len(entries))
	copy(result, entries)

	return result, nil
}

// GetGuildStats returns statistics for a guild's history
func (hm *HistoryManager) GetGuildStats(guildID string) map[string]interface{} {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	history, exists := hm.histories[guildID]
	if !exists {
		return map[string]interface{}{
			"total_songs": 0,
			"oldest_entry": nil,
			"newest_entry": nil,
		}
	}

	stats := map[string]interface{}{
		"total_songs": len(history.Entries),
		"last_updated": history.LastUpdated,
	}

	if len(history.Entries) > 0 {
		stats["newest_entry"] = history.Entries[0].PlayedAt
		stats["oldest_entry"] = history.Entries[len(history.Entries)-1].PlayedAt
	}

	return stats
}

// GetAllGuilds returns a list of all guilds with history
func (hm *HistoryManager) GetAllGuilds() []string {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	guilds := make([]string, 0, len(hm.histories))
	for guildID := range hm.histories {
		guilds = append(guilds, guildID)
	}

	return guilds
}

// ClearGuildHistory clears the history for a specific guild
func (hm *HistoryManager) ClearGuildHistory(guildID string) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if _, exists := hm.histories[guildID]; exists {
		delete(hm.histories, guildID)
		log.Printf("INFO: Cleared history for guild %s", guildID)

		// Auto-save if enabled
		if hm.autosave {
			go func() {
				if err := hm.SaveHistory(); err != nil {
					log.Printf("WARN: Failed to auto-save after clearing history: %v", err)
				}
			}()
		}
	}

	return nil
}

// SaveHistory saves the history data to disk
func (hm *HistoryManager) SaveHistory() error {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(hm.dataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(hm.histories, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(hm.dataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	log.Printf("INFO: Saved history data to %s", hm.dataFile)
	return nil
}

// LoadHistory loads the history data from disk  
func (hm *HistoryManager) LoadHistory() error {
	// Check if file exists
	if _, err := os.Stat(hm.dataFile); os.IsNotExist(err) {
		log.Printf("INFO: History file doesn't exist yet: %s", hm.dataFile)
		return nil // Not an error, just no data yet
	}

	// Read file
	data, err := os.ReadFile(hm.dataFile)
	if err != nil {
		return fmt.Errorf("failed to read history file: %w", err)
	}

	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	// Deserialize from JSON
	if err := json.Unmarshal(data, &hm.histories); err != nil {
		return fmt.Errorf("failed to unmarshal history data: %w", err)
	}

	// Count total entries loaded
	totalEntries := 0
	for _, history := range hm.histories {
		totalEntries += len(history.Entries)
	}

	log.Printf("INFO: Loaded history data from %s (%d guilds, %d total entries)", 
		hm.dataFile, len(hm.histories), totalEntries)
	return nil
}

// CleanupOldEntries removes entries older than the specified duration
func (hm *HistoryManager) CleanupOldEntries(maxAge time.Duration) (int, error) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	totalRemoved := 0

	for guildID, history := range hm.histories {
		originalCount := len(history.Entries)
		
		// Filter out old entries
		filtered := make([]HistoryEntry, 0, originalCount)
		for _, entry := range history.Entries {
			if entry.PlayedAt.After(cutoff) {
				filtered = append(filtered, entry)
			}
		}

		history.Entries = filtered
		removed := originalCount - len(filtered)
		totalRemoved += removed

		if removed > 0 {
			history.LastUpdated = time.Now()
			log.Printf("INFO: Cleaned up %d old history entries for guild %s", removed, guildID)
		}
	}

	// Auto-save if enabled and entries were removed
	if totalRemoved > 0 && hm.autosave {
		go func() {
			if err := hm.SaveHistory(); err != nil {
				log.Printf("WARN: Failed to auto-save after cleanup: %v", err)
			}
		}()
	}

	return totalRemoved, nil
}

// GetRecentlyPlayedUsers returns users who have recently queued songs
func (hm *HistoryManager) GetRecentlyPlayedUsers(guildID string, limit int) []string {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	history, exists := hm.histories[guildID]
	if !exists {
		return []string{}
	}

	// Track unique users in order
	seen := make(map[string]bool)
	users := make([]string, 0, limit)

	for _, entry := range history.Entries {
		if !seen[entry.Song.User] {
			users = append(users, entry.Song.User)
			seen[entry.Song.User] = true

			if len(users) >= limit {
				break
			}
		}
	}

	return users
}

// FindSimilarSongs finds songs in history similar to the given title
func (hm *HistoryManager) FindSimilarSongs(guildID, title string, limit int) []HistoryEntry {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	history, exists := hm.histories[guildID]
	if !exists {
		return []HistoryEntry{}
	}

	var matches []HistoryEntry
	title = strings.ToLower(title)

	for _, entry := range history.Entries {
		entryTitle := strings.ToLower(entry.Song.Title)
		if strings.Contains(entryTitle, title) || strings.Contains(title, entryTitle) {
			matches = append(matches, entry)
			if len(matches) >= limit {
				break
			}
		}
	}

	return matches
}

// Global history manager instance (initialized in main.go)
var historyManager *HistoryManager