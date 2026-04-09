package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/scottstg/yolo/victron"
)

// ─── Victron Tool Implementation ────────

type victronScanResult struct {
	Address    string `json:"address"`
	Name       string `json:"name"`
	RSSI       int    `json:"rssi"`
	IsVictron  bool   `json:"is_victron"`
	DeviceType string `json:"device_type,omitempty"`
}

type victronValue struct {
	Key        string  `json:"key"`
	RawValue   string  `json:"raw_value"`
	FloatValue float64 `json:"float_value"`
	Timestamp  string  `json:"timestamp"`
	Name       string  `json:"name,omitempty"`
	Unit       string  `json:"unit,omitempty"`
}

// Value key metadata for human-readable output
var victronKeyMetadata = map[string]struct {
	Name string
	Unit string
}{
	"V":    {"System Voltage", "V"},
	"A":    {"System Current", "A"},
	"W":    {"System Power", "W"},
	"P.V":  {"PV Input Voltage 1", "V"},
	"P.A":  {"PV Input Current 1", "A"},
	"P.W":  {"PV Input Power 1", "W"},
	"Pi.V": {"PV Input Voltage 2", "V"},
	"Pi.A": {"PV Input Current 2", "A"},
	"Pi.W": {"PV Input Power 2", "W"},
	"SoC":  {"State of Charge", "%"},
	"T":    {"Temperature", "°C"},
	"B.V":  {"Battery Voltage", "V"},
	"B.A":  {"Battery Current", "A"},
	"B.Eh": {"Yield Total (History)", "Ah"},
	"B.En": {"Energy Yield Today", "Ah"},
	"B.Fn": {"Battery Current (Float)", "A"},
	"R":    {"Bulk/Absorb Voltage", "V"},
}

type victronDeviceInfo struct {
	Address      string   `json:"address"`
	Name         string   `json:"name"`
	DeviceType   string   `json:"device_type"`
	Connected    bool     `json:"connected"`
	Capabilities []string `json:"capabilities,omitempty"`
}

func (t *ToolExecutor) victron(args map[string]any) string {
	action := getStringArg(args, "action", "")
	if action == "" {
		return fmt.Sprintf(`{"status":"error","message":"'action' parameter is required. Options: 'scan', 'connect', 'disconnect', 'get_values', 'subscribe', 'device_info'"}`)
	}

	switch action {
	case "scan":
		return t.victronScan(args)
	case "connect":
		return t.victronConnect(args)
	case "disconnect":
		return t.victronDisconnect(args)
	case "get_values":
		return t.victronGetValues(args)
	case "subscribe":
		return t.victronSubscribe(args)
	case "device_info":
		return t.victronDeviceInfo(args)
	default:
		return fmt.Sprintf(`{"status":"error","message":"unknown action '%s'. Options: 'scan', 'connect', 'disconnect', 'get_values', 'subscribe', 'device_info'"}`, action)
	}
}

// Global state for managing connections
var (
	victronClient    *victron.Client
	connectedDevices = make(map[string]*victron.Device)
	clientInitDone   bool
)

func ensureVictronClient() {
	if !clientInitDone || victronClient == nil {
		victronClient = victron.NewClient()

		// Initialize the platform-specific BLE backend
		// This is done lazily to avoid errors on non-supported systems
		victron.InitializeBackend()

		clientInitDone = true
	}
}

func (t *ToolExecutor) victronScan(args map[string]any) string {
	ensureVictronClient()

	durationStr := getStringArg(args, "duration", "10")
	durationSec, err := time.ParseDuration(durationStr + "s")
	if err != nil {
		durationSec = 10 * time.Second
	}
	if durationSec > 60*time.Second {
		durationSec = 60 * time.Second
	}

	results, err := victronClient.Scan(durationSec)
	if err != nil {
		return fmt.Sprintf(`{"status":"error","message":"scan failed: %v"}`, err)
	}

	devices := make([]victronScanResult, len(results))
	for i, d := range results {
		deviceType := inferDeviceType(d.Name)
		devices[i] = victronScanResult{
			Address:    d.Address,
			Name:       d.Name,
			RSSI:       d.RSSI,
			IsVictron:  d.IsVictron,
			DeviceType: deviceType,
		}
	}

	if len(devices) == 0 {
		return fmt.Sprintf(`{"status":"success","message":"No Victron devices found in scan"}`)
	}

	resp := map[string]any{
		"status":  "success",
		"message": fmt.Sprintf("Found %d device(s)", len(devices)),
		"devices": devices,
	}
	return formatJSON(resp)
}

// inferDeviceType determines device type from name string
func inferDeviceType(name string) string {
	nameLower := strings.ToLower(name)
	if strings.Contains(nameLower, "smartsolar") || strings.Contains(nameLower, "mppt") {
		return "SmartSolar"
	}
	if strings.Contains(nameLower, "smartshunt") {
		return "SmartShunt"
	}
	if strings.Contains(nameLower, "ve.direct") || strings.Contains(nameLower, "vedirect") {
		return "VEDirectAdapter"
	}
	if strings.Contains(nameLower, "victron") {
		return "UnknownVictron"
	}
	return "Unknown"
}

func (t *ToolExecutor) victronConnect(args map[string]any) string {
	ensureVictronClient()

	address := getStringArg(args, "address", "")
	if address == "" {
		return fmt.Sprintf(`{"status":"error","message":"'address' parameter is required for connect action. Use 'scan' first to find devices."}`)
	}

	// Parse timeout parameter
	timeoutStr := getStringArg(args, "timeout", "10")
	timeoutSec, err := time.ParseDuration(timeoutStr + "s")
	if err != nil {
		timeoutSec = 10 * time.Second
	}
	if timeoutSec > 30*time.Second {
		timeoutSec = 30 * time.Second
	}

	// Create device and connect
	device, err := victronClient.Connect(address)
	if err != nil {
		return fmt.Sprintf(`{"status":"error","message":"could not prepare connection: %v"}`, err)
	}

	// Use WaitForConnection with timeout instead of direct Connect
	ctx, cancel := context.WithTimeout(context.Background(), timeoutSec)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		err := device.Connect()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Sprintf(`{"status":"error","message":"connection failed: %v"}`, err)
		}
	case <-ctx.Done():
		device.Disconnect()
		return fmt.Sprintf(`{"status":"error","message":"connection timed out after %v"}`, timeoutSec)
	}

	// Store the connected device
	connectedDevices[address] = device

	typeStr := device.Type.String()
	resp := map[string]any{
		"status":  "success",
		"message": fmt.Sprintf("Connected to %s (%s)", address, typeStr),
		"device_info": victronDeviceInfo{
			Address:    address,
			Name:       device.Name,
			DeviceType: typeStr,
			Connected:  true,
		},
	}
	return formatJSON(resp)
}

func (t *ToolExecutor) victronDisconnect(args map[string]any) string {
	address := getStringArg(args, "address", "")

	if address != "" && connectedDevices[address] != nil {
		device := connectedDevices[address]
		device.Disconnect()
		delete(connectedDevices, address)
		resp := map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("Disconnected from %s", address),
		}
		return formatJSON(resp)
	}

	// If no address or not connected, disconnect all
	count := 0
	for addr, device := range connectedDevices {
		device.Disconnect()
		delete(connectedDevices, addr)
		count++
	}

	resp := map[string]any{
		"status":  "success",
		"message": fmt.Sprintf("Disconnected from %d device(s)", count),
	}
	return formatJSON(resp)
}

func (t *ToolExecutor) victronGetValues(args map[string]any) string {
	address := getStringArg(args, "address", "")

	device, connected := connectedDevices[address]
	if !connected || address == "" {
		return fmt.Sprintf(`{"status":"error","message":"device is not connected. Use 'connect' action first with the device address."}`)
	}

	valuesCh := device.Subscribe()

	timeoutStr := getStringArg(args, "timeout", "5")
	timeoutSec, _ := time.ParseDuration(timeoutStr + "s")

	var values []victronValue
	done := time.After(timeoutSec)

	for {
		select {
		case val := <-valuesCh:
			victronVal := victronValue{
				Key:        val.Key,
				RawValue:   val.RawValue,
				FloatValue: val.FloatValue,
				Timestamp:  val.Timestamp.Format(time.RFC3339),
			}
			values = append(values, enrichValue(victronVal))
		case <-done:
			if len(values) == 0 {
				return fmt.Sprintf(`{"status":"success","message":"No values received within timeout"}`)
			}
			resp := map[string]any{
				"status":  "success",
				"message": fmt.Sprintf("Received %d value(s)", len(values)),
				"values":  values,
			}
			return formatJSON(resp)
		}
	}
}

func (t *ToolExecutor) victronSubscribe(args map[string]any) string {
	address := getStringArg(args, "address", "")

	device, connected := connectedDevices[address]
	if !connected || address == "" {
		return fmt.Sprintf(`{"status":"error","message":"device is not connected. Use 'connect' action first with the device address."}`)
	}

	valuesCh := device.Subscribe()

	timeoutStr := getStringArg(args, "timeout", "10")
	timeoutSec, _ := time.ParseDuration(timeoutStr + "s")

	var values []victronValue
	done := time.After(timeoutSec)
	count := 0

	for {
		select {
		case val := <-valuesCh:
			victronVal := victronValue{
				Key:        val.Key,
				RawValue:   val.RawValue,
				FloatValue: val.FloatValue,
				Timestamp:  val.Timestamp.Format(time.RFC3339),
			}
			values = append(values, enrichValue(victronVal))
			count++
		case <-done:
			resp := map[string]any{
				"status":    "success",
				"message":   fmt.Sprintf("Monitored %s for %v, received %d value(s)", address, timeoutSec, count),
				"device_id": address,
				"values":    values,
			}
			return formatJSON(resp)
		}
	}
}

func (t *ToolExecutor) victronDeviceInfo(args map[string]any) string {
	address := getStringArg(args, "address", "")

	device, connected := connectedDevices[address]
	if !connected || address == "" {
		return fmt.Sprintf(`{"status":"error","message":"device is not connected. Use 'connect' action first with the device address."}`)
	}

	typeStr := device.Type.String()
	resp := map[string]any{
		"status":  "success",
		"message": "Device information retrieved",
		"device_info": victronDeviceInfo{
			Address:    device.Address,
			Name:       device.Name,
			DeviceType: typeStr,
			Connected:  true,
		},
	}
	return formatJSON(resp)
}

func formatJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"status":"error","message":"failed to format JSON: %v"}`, err)
	}
	return string(b)
}

// enrichValue adds name and unit information from metadata
func enrichValue(val victronValue) victronValue {
	if meta, ok := victronKeyMetadata[val.Key]; ok {
		val.Name = meta.Name
		val.Unit = meta.Unit
	}
	return val
}

// getAvailableKeys returns a list of all supported value keys with descriptions
func getAvailableKeys() map[string]struct{ Name, Unit string } {
	return victronKeyMetadata
}
