package victron

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestIntegration_MockBackend tests the full flow with mock backend
func TestIntegration_MockBackend(t *testing.T) {
	// Set up mock backend
	mockBackend := NewMockBackend()
	SetBackend(mockBackend)

	// Initialize backend
	err := mockBackend.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize backend: %v", err)
	}
	defer mockBackend.Close()

	// Create client (reserved for future use)
	_ = NewClient()

	// Scan for devices using backend
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	devices, err := mockBackend.ScanForDevices(ctx, 1*time.Second)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(devices) == 0 {
		t.Fatal("Expected to find devices during scan")
	}

	t.Logf("Found %d devices:", len(devices))
	for i, dev := range devices {
		t.Logf("  %d: %s (%s, RSSI: %ddBm)", i+1, dev.Name, dev.Address, dev.RSSI)
	}

	// Connect to first device
	if len(devices) > 0 {
		conn, err := mockBackend.Connect(devices[0].Address)
		if err != nil {
			t.Fatalf("Connection failed: %v", err)
		}
		defer conn.Close()

		if !conn.IsConnected() {
			t.Error("Connection should be connected")
		}

		// Discover services
		services, err := mockBackend.DiscoverServices(conn)
		if err != nil {
			t.Fatalf("DiscoverServices failed: %v", err)
		}

		if len(services) == 0 {
			t.Fatal("Expected to discover at least one service")
		}

		t.Logf("Discovered %d services", len(services))

		// Discover characteristics
		if len(services) > 0 {
			characteristics, err := mockBackend.DiscoverCharacteristics(services[0])
			if err != nil {
				t.Fatalf("DiscoverCharacteristics failed: %v", err)
			}

			if len(characteristics) == 0 {
				t.Fatal("Expected to discover at least one characteristic")
			}

			t.Logf("Discovered %d characteristics in service", len(characteristics))

			// Subscribe to notifications
			if len(characteristics) > 0 {
				notifChan, err := conn.SubscribeToNotifications(characteristics[0])
				if err != nil {
					t.Fatalf("Subscribe failed: %v", err)
				}

				// Read incoming notifications
				timeout := time.After(3 * time.Second)
				valueCount := 0

				for {
					select {
					case data := <-notifChan:
						valueCount++
						t.Logf("Received notification %d: %s", valueCount, string(data))

						// Parse the VE.Direct message
						frame, err := ParseVEDirectMessage(string(data))
						if err != nil {
							t.Logf("Parse warning (may be expected): %v", err)
							continue
						}

						if frame.Valid && frame.DataKey != "" {
							t.Logf("  Parsed: Key=%s, Value=%.3f", frame.DataKey, frame.Value)
						}

						if valueCount >= 3 {
							goto done
						}

					case <-timeout:
						goto done
					}
				}

			done:
				if valueCount == 0 {
					t.Error("Expected to receive at least one notification")
				} else {
					t.Logf("Successfully received %d values", valueCount)
				}

				// Unsubscribe
				err = conn.UnsubscribeFromNotifications(characteristics[0])
				if err != nil {
					t.Errorf("Unsubscribe failed: %v", err)
				}
			}
		}
	}
}

// TestIntegration_ClientWithMockBackend tests the client with mock backend
func TestIntegration_ClientWithMockBackend(t *testing.T) {
	mockBackend := NewMockBackend()

	// Connect directly via mock backend
	conn, err := mockBackend.Connect("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read a value
	data, err := conn.ReadValue(BLECharacteristic{UUID: "test"})
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Parse the received data
	frame, err := ParseVEDirectMessage(string(data))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !frame.Valid {
		t.Error("Parsed frame should be valid")
	}

	if frame.DataKey != "V" {
		t.Errorf("Expected key 'V', got '%s'", frame.DataKey)
	}

	if frame.Value != 12.345 {
		t.Errorf("Expected value 12.345, got %f", frame.Value)
	}

	t.Logf("Successfully read and parsed: %s = %.3f", frame.DataKey, frame.Value)
}

// TestIntegration_ParallelConnections tests handling multiple connections
func TestIntegration_ParallelConnections(t *testing.T) {
	mockBackend := NewMockBackend()

	const numDevices = 5

	var conns []BLEConnection

	// Create multiple connections
	for i := 0; i < numDevices; i++ {
		addr := fmt.Sprintf("AA:BB:CC:DD:EE:%02X", i+1)
		conn, err := mockBackend.Connect(addr)
		if err != nil {
			t.Fatalf("Connection %d failed: %v", i, err)
		}
		conns = append(conns, conn)
	}

	// All connections should be active
	for i, conn := range conns {
		if !conn.IsConnected() {
			t.Errorf("Connection %d should be connected", i)
		}
	}

	// Read from all connections concurrently
	done := make(chan bool, numDevices)

	for i, conn := range conns {
		go func(idx int, c BLEConnection) {
			data, err := c.ReadValue(BLECharacteristic{UUID: "test"})
			if err != nil {
				t.Logf("Connection %d read failed: %v", idx, err)
				done <- false
				return
			}

			if len(data) == 0 {
				t.Logf("Connection %d returned empty data", idx)
				done <- false
				return
			}

			done <- true
		}(i, conn)
	}

	// Wait for all reads to complete
	successCount := 0
	for i := 0; i < numDevices; i++ {
		if <-done {
			successCount++
		}
	}

	t.Logf("Successfully read from %d/%d connections", successCount, numDevices)

	// Clean up
	for _, conn := range conns {
		conn.Close()
	}
}

// TestDevice_SmartSolarValues tests parsing typical SmartSolar values
func TestIntegration_SmartSolarValues(t *testing.T) {
	// Typical SmartSolar messages
	messages := []string{
		"/V(V=13.500)",     // System voltage
		"/A(A=2.500)",      // Charging current
		"/W(W=33.750)",     // Charging power
		"/P.V(P.V=18.200)", // PV input voltage
		"/P.A(P.A=4.800)",  // PV input current
		"/P.W(P.W=87.360)", // PV input power
	}

	for _, msg := range messages {
		fullMsg := msg + string(calculateChecksum([]byte(msg)))
		frame, err := ParseVEDirectMessage(fullMsg)

		if err != nil {
			t.Fatalf("Failed to parse %q: %v", msg, err)
		}

		if !frame.Valid {
			t.Errorf("Frame should be valid for message: %q", msg)
		}

		t.Logf("Parsed: Key=%s, Value=%.3f", frame.DataKey, frame.Value)

		// Verify we got the right key and value type
		info := GetValueType(frame.DataKey)
		if info != nil {
			t.Logf("  Type: %s (%s)", info.Name, info.Unit)
		}
	}
}

// TestDevice_SmartShuntValues tests parsing typical SmartShunt values
func TestIntegration_SmartShuntValues(t *testing.T) {
	// Typical SmartShunt messages
	messages := []string{
		"/V(V=12.650)",     // Battery voltage
		"/B.A(A=-5.200)",   // Battery current (discharging)
		"/SoC(SoC=78.500)", // State of charge
	}

	for _, msg := range messages {
		fullMsg := msg + string(calculateChecksum([]byte(msg)))
		frame, err := ParseVEDirectMessage(fullMsg)

		if err != nil {
			t.Fatalf("Failed to parse %q: %v", msg, err)
		}

		if !frame.Valid {
			t.Errorf("Frame should be valid for message: %q", msg)
		}

		t.Logf("Parsed SmartShunt: Key=%s, Value=%.3f", frame.DataKey, frame.Value)
	}
}
