package victron

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistoricalStore_New(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewHistoricalStore(tmpDir)
	
	if store == nil {
		t.Fatal("Expected non-nil store")
	}
	
	if store.data == nil {
		t.Error("Expected initialized data map")
	}
	
	if _, exists := store.data[""]; exists {
		t.Error("Expected empty data map on new store")
	}
}

func TestHistoricalStore_RecordAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewHistoricalStore(tmpDir)
	
	address := "AA:BB:CC:DD:EE:FF"
	key := "V.A"
	
	// Record some data points
	for i := 0; i < 5; i++ {
		store.Record(address, key, float64(14.0+float64(i)*0.1), "V")
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}
	
	// Read back the data
	readings := store.GetReadings(address, key, 0)
	
	if len(readings) != 5 {
		t.Errorf("Expected 5 readings, got %d", len(readings))
	}
	
	for i, reading := range readings {
		if reading.Address != address {
			t.Errorf("Entry %d: expected address %s, got %s", i, address, reading.Address)
		}
		
		if reading.Key != key {
			t.Errorf("Entry %d: expected key %s, got %s", i, key, reading.Key)
		}
		
		expectedValue := 14.0 + float64(i)*0.1
		if reading.Value != expectedValue {
			t.Errorf("Entry %d: expected value %.1f, got %.1f", i, expectedValue, reading.Value)
		}
	}
}

func TestHistoricalStore_Limit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewHistoricalStore(tmpDir)
	
	address := "AA:BB:CC:DD:EE:FF"
	key := "A"
	
	// Record 10 data points
	for i := 0; i < 10; i++ {
		store.Record(address, key, float64(i+1), "A")
	}
	
	// Request only last 3
	readings := store.GetReadings(address, key, 3)
	
	if len(readings) != 3 {
		t.Errorf("Expected 3 readings with limit, got %d", len(readings))
	}
	
	// Should return the last 3 values (8, 9, 10)
	expectedValues := []float64{8, 9, 10}
	for i, reading := range readings {
		if reading.Value != expectedValues[i] {
			t.Errorf("Entry %d: expected value %.0f, got %.0f", i, expectedValues[i], reading.Value)
		}
	}
}

func TestHistoricalStore_MultipleKeys(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewHistoricalStore(tmpDir)
	
	address := "AA:BB:CC:DD:EE:FF"
	
	// Record for multiple keys
	keys := []string{"V.A", "A", "P.W"}
	for _, k := range keys {
		for i := 0; i < 3; i++ {
			store.Record(address, k, float64(i+1), "")
		}
	}
	
	// Verify each key has correct data
	for _, k := range keys {
		readings := store.GetReadings(address, k, 0)
		if len(readings) != 3 {
			t.Errorf("Key %s: expected 3 readings, got %d", k, len(readings))
		}
	}
}

func TestHistoricalStore_GetReadingsForDevice(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewHistoricalStore(tmpDir)
	
	address1 := "AA:BB:CC:DD:EE:FF"
	address2 := "11:22:33:44:55:66"
	
	// Record for two devices
	store.Record(address1, "V.A", 14.0, "V")
	store.Record(address1, "A", 5.0, "A")
	store.Record(address2, "B.A", 13.0, "V")
	
	// Get all readings for device 1
	readings := store.GetReadingsForDevice(address1, 0)
	
	if len(readings) != 2 {
		t.Errorf("Expected readings for 2 keys, got %d", len(readings))
	}
	
	if _, exists := readings["V.A"]; !exists {
		t.Error("Expected V.A key in results")
	}
	
	if _, exists := readings["A"]; !exists {
		t.Error("Expected A key in results")
	}
	
	// Should not include device 2's data
	if _, exists := readings["B.A"]; exists {
		t.Error("Should not include B.A from different device")
	}
}

func TestHistoricalStore_Statistics(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewHistoricalStore(tmpDir)
	
	address := "AA:BB:CC:DD:EE:FF"
	key := "V.A"
	
	// Record values: 10, 20, 30, 40, 50
	values := []float64{10, 20, 30, 40, 50}
	for _, v := range values {
		store.Record(address, key, v, "V")
	}
	
	stats := store.GetStatistics(address, key)
	
	if stats == nil {
		t.Fatal("Expected non-nil statistics")
	}
	
	expectedStats := map[string]float64{
		"count": 5,
		"sum":   150,
		"avg":   30,
		"min":   10,
		"max":   50,
	}
	
	for k, expected := range expectedStats {
		if stats[k] != expected {
			t.Errorf("Statistic %s: expected %.1f, got %.1f", k, expected, stats[k])
		}
	}
}

func TestHistoricalStore_EmptyStatistics(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewHistoricalStore(tmpDir)
	
	stats := store.GetStatistics("AA:BB:CC:DD:EE:FF", "V.A")
	
	if stats != nil {
		t.Error("Expected nil statistics for non-existent key")
	}
}

func TestHistoricalStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	store1 := NewHistoricalStore(tmpDir)
	
	address := "AA:BB:CC:DD:EE:FF"
	key := "V.A"
	
	// Record and save data
	for i := 0; i < 20; i++ {
		store1.Record(address, key, float64(i+1), "V")
	}
	
	// Create new store instance (simulates restart)
	store2 := NewHistoricalStore(tmpDir)
	
	// Data should be loaded from disk
	readings := store2.GetReadings(address, key, 0)
	if len(readings) != 20 {
		t.Errorf("Expected 20 persisted readings, got %d", len(readings))
	}
	
	// Verify data integrity
	for i, reading := range readings {
		expectedValue := float64(i + 1)
		if reading.Value != expectedValue {
			t.Errorf("Entry %d: expected value %.0f, got %.0f", i, expectedValue, reading.Value)
		}
	}
}

func TestHistoricalStore_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewHistoricalStore(tmpDir)
	
	address := "AA:BB:CC:DD:EE:FF"
	key := "V.A"
	
	// Record some data (need 10+ entries to trigger save)
	for i := 0; i < 10; i++ {
		store.Record(address, key, float64(i+1), "V")
	}
	
	// Clear all data
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	
	// Verify data is gone
	readings := store.GetReadings(address, key, 0)
	if len(readings) != 0 {
		t.Errorf("Expected no readings after clear, got %d", len(readings))
	}
	
	// File should be deleted
	filePath := filepath.Join(tmpDir, "victron_history.json")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Expected history file to be deleted")
	}
}

func TestHistoricalStore_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewHistoricalStore(tmpDir)
	
	address := "AA:BB:CC:DD:EE:FF"
	key := "V.A"
	
	writerDone := make(chan bool)
	readerDone := make(chan bool)
	
	// Concurrent writers
	go func() {
		defer close(writerDone)
		for i := 0; i < 100; i++ {
			store.Record(address, key, float64(i), "V")
		}
	}()
	
	// Concurrent readers
	go func() {
		defer close(readerDone)
		for i := 0; i < 100; i++ {
			_ = store.GetReadings(address, key, 10)
		}
	}()
	
	<-writerDone
	<-readerDone
	
	// Verify data integrity
	readings := store.GetReadings(address, key, 0)
	if len(readings) != 100 {
		t.Errorf("Expected 100 readings, got %d", len(readings))
	}
}
