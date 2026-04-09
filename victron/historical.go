// Package victron provides Bluetooth Low Energy communication with Victron Energy devices
package victron

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HistoricalEntry represents a single data point with timestamp
type HistoricalEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Address   string    `json:"address"` // Device MAC address
	Key       string    `json:"key"`     // VE.Direct key (e.g., "V.A", "A")
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"` // Optional unit identifier
}

// HistoricalStore manages persistent storage of device readings
type HistoricalStore struct {
	mu       sync.RWMutex
	data     map[string][]HistoricalEntry // key = address+key combination
	dataDir  string
	fileName string
}

// NewHistoricalStore creates a new historical data store
func NewHistoricalStore(dataDir string) *HistoricalStore {
	store := &HistoricalStore{
		data:     make(map[string][]HistoricalEntry),
		dataDir:  dataDir,
		fileName: "victron_history.json",
	}

	// Load existing data if available
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		// Silently ignore load errors - we'll start fresh
		_ = err
	}

	return store
}

// Record adds a new historical entry for a device value
func (s *HistoricalStore) Record(address, key string, value float64, unit string) {
	s.mu.Lock()

	entry := HistoricalEntry{
		Timestamp: time.Now(),
		Address:   address,
		Key:       key,
		Value:     value,
		Unit:      unit,
	}

	keyID := address + ":" + key
	s.data[keyID] = append(s.data[keyID], entry)

	// Check if we should save (every 10 entries for this key)
	shouldSave := len(s.data[keyID])%10 == 0
	s.mu.Unlock()

	// Save periodically outside the lock to avoid deadlock
	if shouldSave {
		_ = s.save() // Ignore save errors in hot path
	}
}

// GetReadings returns all readings for a specific device and key
func (s *HistoricalStore) GetReadings(address, key string, limit int) []HistoricalEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keyID := address + ":" + key
	entries, exists := s.data[keyID]
	if !exists || len(entries) == 0 {
		return nil
	}

	// Return last N entries (or all if limit <= 0)
	if limit > 0 && len(entries) > limit {
		result := make([]HistoricalEntry, limit)
		start := len(entries) - limit
		copy(result, entries[start:])
		return result
	}

	// Return a copy to prevent external modification
	result := make([]HistoricalEntry, len(entries))
	copy(result, entries)
	return result
}

// GetReadingsForDevice returns all readings for a device (all keys)
func (s *HistoricalStore) GetReadingsForDevice(address string, limit int) map[string][]HistoricalEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string][]HistoricalEntry)
	for keyID, entries := range s.data {
		if len(keyID) > len(address)+1 && keyID[:len(address)] == address {
			key := keyID[len(address)+1:]
			if limit > 0 && len(entries) > limit {
				entryCopy := make([]HistoricalEntry, limit)
				start := len(entries) - limit
				copy(entryCopy, entries[start:])
				result[key] = entryCopy
			} else {
				entryCopy := make([]HistoricalEntry, len(entries))
				copy(entryCopy, entries)
				result[key] = entryCopy
			}
		}
	}

	return result
}

// GetStatistics calculates basic statistics for a key
func (s *HistoricalStore) GetStatistics(address, key string) map[string]float64 {
	readings := s.GetReadings(address, key, 0)
	if len(readings) == 0 {
		return nil
	}

	var sum, min, max float64
	min = readings[0].Value

	for _, entry := range readings {
		sum += entry.Value
		if entry.Value < min {
			min = entry.Value
		}
		if entry.Value > max {
			max = entry.Value
		}
	}

	count := float64(len(readings))
	return map[string]float64{
		"count": count,
		"sum":   sum,
		"avg":   sum / count,
		"min":   min,
		"max":   max,
	}
}

// Clear removes all historical data
func (s *HistoricalStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string][]HistoricalEntry)
	filePath := filepath.Join(s.dataDir, s.fileName)
	return os.Remove(filePath)
}

// load reads historical data from disk
func (s *HistoricalStore) load() error {
	filePath := filepath.Join(s.dataDir, s.fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var loadedData map[string][]HistoricalEntry
	if err := json.Unmarshal(data, &loadedData); err != nil {
		return err
	}

	s.mu.Lock()
	s.data = loadedData
	s.mu.Unlock()

	return nil
}

// save writes historical data to disk
func (s *HistoricalStore) save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.dataDir, s.fileName)
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}
