package tts

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheMetadata represents metadata for a specific provider/voice combination
type CacheMetadata struct {
	Version      string       `json:"version"`
	Provider     string       `json:"provider"`
	Voice        string       `json:"voice"`
	CacheEntries []CacheEntry `json:"cache_entries"`
}

// CacheEntry represents a single cached audio file
type CacheEntry struct {
	Hash       string    `json:"hash"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
	FileSize   int64     `json:"file_size"`
	DurationMs int       `json:"duration_ms,omitempty"`
}

// MasterMetadata contains summary information across all providers and voices
type MasterMetadata struct {
	Version       string                  `json:"version"`
	LastUpdated   time.Time               `json:"last_updated"`
	TotalFiles    int                     `json:"total_files"`
	TotalSize     int64                   `json:"total_size"`
	ProviderStats map[string]ProviderStat `json:"provider_stats"`
}

// ProviderStat contains statistics for a specific provider
type ProviderStat struct {
	FileCount  int            `json:"file_count"`
	TotalSize  int64          `json:"total_size"`
	VoiceStats map[string]int `json:"voice_stats"` // voice -> file count
	LastUsed   time.Time      `json:"last_used"`
}

var (
	metadataLocks = sync.Map{} // path -> *sync.Mutex
	masterLock    sync.Mutex
)

// getMetadataLock returns a mutex for the given metadata file path
func getMetadataLock(path string) *sync.Mutex {
	lock, _ := metadataLocks.LoadOrStore(path, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// loadOrCreateMetadata loads existing metadata or creates a new one
func loadOrCreateMetadata(metadataPath, provider, voice string, logger *slog.Logger) *CacheMetadata {
	lock := getMetadataLock(metadataPath)
	lock.Lock()
	defer lock.Unlock()

	if _, err := os.Stat(metadataPath); err == nil {
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			logger.Error("failed to read metadata file", "path", metadataPath, "err", err)
			return newMetadata(provider, voice)
		}

		var metadata CacheMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			logger.Error("failed to unmarshal metadata", "path", metadataPath, "err", err)
			return newMetadata(provider, voice)
		}

		return &metadata
	}

	return newMetadata(provider, voice)
}

// newMetadata creates a new metadata structure
func newMetadata(provider, voice string) *CacheMetadata {
	return &CacheMetadata{
		Version:      "1.0",
		Provider:     provider,
		Voice:        voice,
		CacheEntries: []CacheEntry{},
	}
}

// saveMetadataAtomic saves metadata atomically using a temp file + rename
func saveMetadataAtomic(metadataPath string, metadata *CacheMetadata, logger *slog.Logger) error {
	lock := getMetadataLock(metadataPath)
	lock.Lock()
	defer lock.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(metadataPath), 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to temp file
	tempPath := metadataPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp metadata file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, metadataPath); err != nil {
		os.Remove(tempPath) // Clean up temp file on error
		return fmt.Errorf("failed to rename metadata file: %w", err)
	}

	logger.Info("saved metadata", "path", metadataPath, "entries", len(metadata.CacheEntries))
	return nil
}

// updateMetadata adds a new cache entry to the metadata file
func updateMetadata(provider, voice, hash, text string, fileSize int64, logger *slog.Logger) error {
	baseDir := os.Getenv("AUDIO_DIR")
	if baseDir == "" {
		baseDir = filepath.Join(".", "audio")
	}

	metadataPath := filepath.Join(baseDir, sanitizeDirectoryName(provider), sanitizeDirectoryName(voice), "metadata.json")

	metadata := loadOrCreateMetadata(metadataPath, provider, voice, logger)

	// Check if entry already exists
	for i, entry := range metadata.CacheEntries {
		if entry.Hash == hash {
			// Update existing entry
			metadata.CacheEntries[i].CreatedAt = time.Now()
			metadata.CacheEntries[i].FileSize = fileSize
			logger.Info("updated existing cache entry", "hash", hash)
			if err := saveMetadataAtomic(metadataPath, metadata, logger); err != nil {
				return err
			}
			return updateMasterMetadata(logger)
		}
	}

	metadata.CacheEntries = append(metadata.CacheEntries, CacheEntry{
		Hash:      hash,
		Text:      text,
		CreatedAt: time.Now(),
		FileSize:  fileSize,
	})

	if err := saveMetadataAtomic(metadataPath, metadata, logger); err != nil {
		return err
	}

	return updateMasterMetadata(logger)
}

// loadMasterMetadata loads the master metadata file
func loadMasterMetadata(logger *slog.Logger) *MasterMetadata {
	masterLock.Lock()
	defer masterLock.Unlock()

	baseDir := os.Getenv("AUDIO_DIR")
	if baseDir == "" {
		baseDir = filepath.Join(".", "audio")
	}

	masterPath := filepath.Join(baseDir, "master_metadata.json")

	if _, err := os.Stat(masterPath); err == nil {
		data, err := os.ReadFile(masterPath)
		if err != nil {
			logger.Error("failed to read master metadata", "err", err)
			return newMasterMetadata()
		}

		var master MasterMetadata
		if err := json.Unmarshal(data, &master); err != nil {
			logger.Error("failed to unmarshal master metadata", "err", err)
			return newMasterMetadata()
		}

		return &master
	}

	return newMasterMetadata()
}

// newMasterMetadata creates a new master metadata structure
func newMasterMetadata() *MasterMetadata {
	return &MasterMetadata{
		Version:       "1.0",
		LastUpdated:   time.Now(),
		TotalFiles:    0,
		TotalSize:     0,
		ProviderStats: make(map[string]ProviderStat),
	}
}

// saveMasterMetadata saves the master metadata atomically
func saveMasterMetadata(master *MasterMetadata, logger *slog.Logger) error {
	masterLock.Lock()
	defer masterLock.Unlock()

	baseDir := os.Getenv("AUDIO_DIR")
	if baseDir == "" {
		baseDir = filepath.Join(".", "audio")
	}

	masterPath := filepath.Join(baseDir, "master_metadata.json")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(masterPath), 0755); err != nil {
		return fmt.Errorf("failed to create audio directory: %w", err)
	}

	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(master, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal master metadata: %w", err)
	}

	// Write to temp file
	tempPath := masterPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp master metadata file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, masterPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename master metadata file: %w", err)
	}

	logger.Info("saved master metadata", "total_files", master.TotalFiles, "providers", len(master.ProviderStats))
	return nil
}

// updateMasterMetadata recalculates and updates the master metadata by scanning all provider metadata files
func updateMasterMetadata(logger *slog.Logger) error {
	baseDir := os.Getenv("AUDIO_DIR")
	if baseDir == "" {
		baseDir = filepath.Join(".", "audio")
	}

	master := newMasterMetadata()

	// Walk through all provider/voice directories
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors in walking
		}

		if info.IsDir() || info.Name() != "metadata.json" {
			return nil
		}

		// Read the metadata file
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("failed to read metadata file", "path", path, "err", err)
			return nil
		}

		var metadata CacheMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			logger.Warn("failed to unmarshal metadata file", "path", path, "err", err)
			return nil
		}

		// Update statistics
		provider := metadata.Provider
		voice := metadata.Voice

		stat, exists := master.ProviderStats[provider]
		if !exists {
			stat = ProviderStat{
				VoiceStats: make(map[string]int),
			}
		}

		for _, entry := range metadata.CacheEntries {
			master.TotalFiles++
			master.TotalSize += entry.FileSize
			stat.FileCount++
			stat.TotalSize += entry.FileSize
			stat.VoiceStats[voice]++

			if entry.CreatedAt.After(stat.LastUsed) {
				stat.LastUsed = entry.CreatedAt
			}
		}

		master.ProviderStats[provider] = stat
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk audio directory: %w", err)
	}

	master.LastUpdated = time.Now()
	return saveMasterMetadata(master, logger)
}

// sanitizeDirectoryName sanitizes a string to be used as a directory name
func sanitizeDirectoryName(name string) string {
	return sanitizeFileName(name)
}

// GetCacheStats returns cache statistics from the master metadata
func GetCacheStats(logger *slog.Logger) (*MasterMetadata, error) {
	// Ensure master metadata is up to date
	if err := updateMasterMetadata(logger); err != nil {
		return nil, err
	}

	return loadMasterMetadata(logger), nil
}

// SearchCacheByText searches for cache entries containing the given text
func SearchCacheByText(searchText string, logger *slog.Logger) ([]CacheEntry, error) {
	baseDir := os.Getenv("AUDIO_DIR")
	if baseDir == "" {
		baseDir = filepath.Join(".", "audio")
	}

	var results []CacheEntry

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() || info.Name() != "metadata.json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var metadata CacheMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			return nil
		}

		for _, entry := range metadata.CacheEntries {
			if contains(entry.Text, searchText) {
				results = append(results, entry)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search cache: %w", err)
	}

	return results, nil
}

// contains checks if s contains substr (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
