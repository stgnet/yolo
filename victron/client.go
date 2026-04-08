// Client implementation for connecting to Victron devices via BLE
package victron

import (
	"sync"
	"time"
)

// Client implements BLE client functionality for discovering and connecting to Victron devices
type Client struct {
	mu               sync.RWMutex
	connectedDevices map[string]*Device // Address -> Device mapping
	scanning         bool
	scanResults      []DiscoverableDevice
	scanMutex        sync.Mutex
}

// NewClient creates a new Victron BLE client
func NewClient() *Client {
	return &Client{
		connectedDevices: make(map[string]*Device),
		scanResults:      make([]DiscoverableDevice, 0),
	}
}

// Scan searches for nearby Victron devices advertising over BLE
// duration specifies how long to scan (e.g., 10*time.Second)
func (c *Client) Scan(duration time.Duration) ([]DiscoverableDevice, error) {
	c.scanMutex.Lock()
	defer c.scanMutex.Unlock()
	
	c.scanning = true
	defer func() { c.scanning = false }()
	
	// Start BLE scan - this is where the actual BLE library integration would happen
	// For now, we'll return a timeout indicating no implementation yet
	select {
	case <-time.After(duration):
		return c.scanResults, nil
	}
}

// StopScan stops an ongoing scan
func (c *Client) StopScan() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scanning = false
}

// Connect attempts to connect to a specific Victron device by address
func (c *Client) Connect(address string) (*Device, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if already connected
	if device, exists := c.connectedDevices[address]; exists {
		if device.connected {
			return device, nil
		}
	}
	
	// Create new device connection
	device := &Device{
		Address:      address,
		values:       make(map[string]Value),
		valueChan:    make(chan Value, 100),
		valueUpdates: make(chan map[string]Value, 10),
		stopChan:     make(chan struct{}),
	}
	
	// Actual BLE connection logic would go here
	// This is where we'd use a library like github.com/go-bluetooth/bt
	
	c.connectedDevices[address] = device
	
	return device, nil
}

// Disconnect closes the connection to a device
func (c *Client) Disconnect(address string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if device, exists := c.connectedDevices[address]; exists {
		device.disconnectInternal()
		delete(c.connectedDevices, address)
	}
	
	return nil
}

// GetAllConnected returns all currently connected devices
func (c *Client) GetAllConnected() []*Device {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	devices := make([]*Device, 0, len(c.connectedDevices))
	for _, device := range c.connectedDevices {
		if device.connected {
			devices = append(devices, device)
		}
	}
	return devices
}

// GetDevice retrieves a connected device by address
func (c *Client) GetDevice(address string) (*Device, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	device, exists := c.connectedDevices[address]
	if !exists || !device.connected {
		return nil, false
	}
	return device, true
}

// Discover scans and connects to the first available Victron device
func (c *Client) Discover() (*Device, error) {
	// Scan for devices
	results, err := c.Scan(10 * time.Second)
	if err != nil {
		return nil, err
	}
	
	// Filter for Victron devices
	var victronDevices []DiscoverableDevice
	for _, result := range results {
		if result.IsVictron {
			victronDevices = append(victronDevices, result)
		}
	}
	
	if len(victronDevices) == 0 {
		return nil, &NotFoundError{Query: "victron device"}
	}
	
	// Connect to the strongest signal
	bestDevice := victronDevices[0]
	for _, dev := range victronDevices[1:] {
		if dev.RSSI > bestDevice.RSSI {
			bestDevice = dev
		}
	}
	
	return c.Connect(bestDevice.Address)
}

// Device methods

// Connect establishes connection and starts receiving data
func (d *Device) Connect() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if d.connected {
		return nil // Already connected
	}
	
	// BLE connection logic here
	// After connecting, discover services and characteristics
	// Subscribe to notifications
	
	d.connected = true
	
	// Start goroutine to process incoming values
	go d.processIncomingValues()
	
	return nil
}

// Disconnect closes the device connection
func (d *Device) Disconnect() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.disconnectInternal()
}

func (d *Device) disconnectInternal() {
	if !d.connected {
		return
	}
	
	d.connected = false
	
	// Stop receiving updates
	select {
	case <-d.stopChan:
		// Already closed
	default:
		close(d.stopChan)
	}
}

// processIncomingValues handles the channel of incoming value updates
func (d *Device) processIncomingValues() {
	for {
		select {
		case <-d.stopChan:
			return
		case val := <-d.valueChan:
			d.valuesMutex.Lock()
			d.values[val.Key] = val
			d.valuesMutex.Unlock()
			
			// Send batched update
			select {
			case d.valueUpdates <- d.getValuesCopy():
			default:
				// Channel full, skip this update
			}
		}
	}
}

func (d *Device) getValuesCopy() map[string]Value {
	d.valuesMutex.RLock()
	defer d.valuesMutex.RUnlock()
	
	copy := make(map[string]Value, len(d.values))
	for k, v := range d.values {
		copy[k] = v
	}
	return copy
}

// GetValue returns the latest value for a specific key
func (d *Device) GetValue(key string) (Value, bool) {
	d.valuesMutex.RLock()
	defer d.valuesMutex.RUnlock()
	
	value, exists := d.values[key]
	return value, exists
}

// GetAllValues returns a copy of all current values
func (d *Device) GetAllValues() map[string]Value {
	return d.getValuesCopy()
}

// Subscribe sets up a channel to receive value updates in real-time
func (d *Device) Subscribe() <-chan Value {
	return d.valueChan
}

// SubscribeBatched sets up a channel to receive batched updates
func (d *Device) SubscribeBatched() <-chan map[string]Value {
	return d.valueUpdates
}

// IsConnected checks if the device is currently connected
func (d *Device) IsConnected() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.connected
}

// GetTypeInfo returns information about what this device type supports
func (dt DeviceType) GetSupportedKeys() []string {
	switch dt {
	case DeviceTypeSmartSolar:
		return []string{
			KeySystemVoltage, KeySystemCurrent, KeySystemPower,
			KeyPVInputVoltage1, KeyPVInputCurrent1, KeyPVInputPower1,
			KeyPVInputVoltage2, KeyPVInputCurrent2, KeyPVInputPower2,
			KeyTemperature, KeyErrorCode, KeyAlgorithmState,
		}
	case DeviceTypeSmartShunt:
		return []string{
			KeyBatteryVoltage, KeyBatteryCurrent,
			KeyStateOfCharge, KeyEnergyYieldToday,
			KeySystemVoltage,
		}
	case DeviceTypeVEDirectAdapter:
		// Can support all keys depending on connected device
		return nil
	default:
		return []string{}
	}
}

// ScanWithFilter scans with specific filters to find a particular device type
func (c *Client) ScanWithFilter(filter ScanFilter, timeout time.Duration) ([]DiscoverableDevice, error) {
	c.scanMutex.Lock()
	defer c.scanMutex.Unlock()
	
	c.scanning = true
	defer func() { c.scanning = false }()
	
	// Filter results based on criteria
	var filtered []DiscoverableDevice
	
	for _, result := range c.scanResults {
		if matchesFilter(result, filter) {
			filtered = append(filtered, result)
		}
	}
	
	return filtered, nil
}

func matchesFilter(device DiscoverableDevice, filter ScanFilter) bool {
	if filter.Name != "" && !containsIgnoreCase(device.Name, filter.Name) {
		return false
	}
	return true
}

// WaitForConnection waits for a device to become connected with timeout
func (d *Device) WaitForConnection(timeout time.Duration) error {
	start := time.Now()
	
	for {
		if d.IsConnected() {
			return nil
		}
		
		elapsed := time.Since(start)
		if elapsed >= timeout {
			return &TimeoutError{Operation: "connect", Duration: timeout}
		}
		
		time.Sleep(100 * time.Millisecond)
	}
}
