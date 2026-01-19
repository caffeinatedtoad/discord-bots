package tts

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"path/filepath"
)

// generateHash creates a 12-character hash from the input text
func generateHash(input string) string {
	hash := sha256.Sum256([]byte(input))
	// Use first 6 bytes = 12 hex characters
	return hex.EncodeToString(hash[:6])
}

// getCachePath returns the new hierarchical cache path for a given provider, voice, and input
func getCachePath(provider, voice, input string) string {
	baseDir := os.Getenv("AUDIO_DIR")
	if baseDir == "" {
		baseDir = filepath.Join(".", "audio")
	}

	// Normalize provider and voice names
	providerDir := sanitizeDirectoryName(provider)
	voiceDir := sanitizeDirectoryName(voice)

	// Generate hash
	hash := generateHash(input)

	// Build path: audio/{provider}/{voice}/{hash}.wav
	return filepath.Join(baseDir, providerDir, voiceDir, hash+".wav")
}

// getProviderFromGeneratorName maps generator names to provider names for the cache hierarchy
func getProviderFromGeneratorName(generatorName string) string {
	// Map tiktok to marcus as per requirement
	if generatorName == "tiktok" {
		return "marcus"
	}
	return generatorName
}

// getCachePathForGenerator gets the cache path using the generator's name as provider
func getCachePathForGenerator(generator Generator, voice, input string) string {
	provider := getProviderFromGeneratorName(generator.Name())
	return getCachePath(provider, voice, input)
}

// getFileNameWithFallback tries new cache path first, then falls back to legacy if it exists
func getFileNameWithFallback(provider, voice, input string, logger *slog.Logger) string {
	// Try new hash-based path first
	newPath := getCachePath(provider, voice, input)
	if fileIsCached(newPath) {
		logger.Info("found file in new cache structure", "path", newPath)
		return newPath
	}

	// Fall back to legacy flat structure
	legacyPath := legacyFileName(input)
	if fileIsCached(legacyPath) {
		logger.Info("found file in legacy cache structure", "path", legacyPath)
		return legacyPath
	}

	// Also check voice-prefixed legacy format
	voicePrefixedLegacy := getFileName(input, voice)
	if fileIsCached(voicePrefixedLegacy) {
		logger.Info("found file in voice-prefixed legacy cache", "path", voicePrefixedLegacy)
		return voicePrefixedLegacy
	}

	// Return new path for creation
	logger.Info("no cached file found", "path", newPath)
	return newPath
}

// ensureCacheDirectory creates the cache directory structure if it doesn't exist
func ensureCacheDirectory(provider, voice string) error {
	baseDir := os.Getenv("AUDIO_DIR")
	if baseDir == "" {
		baseDir = filepath.Join(".", "audio")
	}

	providerDir := sanitizeDirectoryName(provider)
	voiceDir := sanitizeDirectoryName(voice)

	dir := filepath.Join(baseDir, providerDir, voiceDir)
	return os.MkdirAll(dir, 0755)
}
