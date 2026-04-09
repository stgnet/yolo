// Package victron provides functionality to connect to and read values from
// Victron Energy devices via Bluetooth Low Energy (BLE) GATT protocol.
//
// This library supports:
// - Victron SmartSolar MPPT charge controllers
// - Victron SmartShunt battery monitors
// - Victron VE.Direct to Bluetooth Smart adapters
// - Other Victron BLE-enabled devices
package victron

import (
	"fmt"
	"sync"
	"time"
)

// DeviceType represents the type of Victron device
type DeviceType int

const (
	DeviceTypeUnknown         DeviceType = iota
	DeviceTypeSmartSolar                 // MPPT charge controller
	DeviceTypeSmartShunt                 // Battery monitor
	DeviceTypeVEDirectAdapter            // VE.Direct to Bluetooth adapter
	DeviceTypePhoenixInverter            // Phoenix inverter/charger
	DeviceTypeMultiPlus                  // MultiPlus inverter/charger
)

func (dt DeviceType) String() string {
	switch dt {
	case DeviceTypeSmartSolar:
		return "SmartSolar"
	case DeviceTypeSmartShunt:
		return "SmartShunt"
	case DeviceTypeVEDirectAdapter:
		return "VE.Direct Adapter"
	case DeviceTypePhoenixInverter:
		return "Phoenix Inverter"
	case DeviceTypeMultiPlus:
		return "MultiPlus"
	default:
		return "Unknown"
	}
}

// DeviceInfo contains information about a discovered Victron device
type DeviceInfo struct {
	Address        string     // MAC address or Bluetooth address
	Name           string     // Device name from advertisement
	Type           DeviceType // Device type
	SignalStrength int        // RSSI value (negative, closer to 0 is stronger)
}

// Value represents a data point read from a Victron device
type Value struct {
	Key        string    // Data key (e.g., "V", "A", "P")
	RawValue   string    // Raw string value from device
	FloatValue float64   // Parsed float value
	Timestamp  time.Time // When the value was received
}

// Device represents a connected Victron device
type Device struct {
	Address      string
	Name         string
	Type         DeviceType
	connected    bool
	values       map[string]Value
	valuesMutex  sync.RWMutex
	valueChan    chan Value
	valueUpdates chan map[string]Value
	stopChan     chan struct{}
	mu           sync.Mutex
}

// Error types
type (
	ConnectionError struct {
		Address string
		Err     error
	}

	NotFoundError struct {
		Query string
	}

	TimeoutError struct {
		Operation string
		Duration  time.Duration
	}

	ParseError struct {
		Field string
		Raw   string
		Err   error
	}
)

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection error to %s: %v", e.Address, e.Err)
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("no device found matching: %s", e.Query)
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("%s timed out after %v", e.Operation, e.Duration)
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error for field %s (value: %s): %v", e.Field, e.Raw, e.Err)
}

// Common data keys used by Victron devices
const (
	// Voltage keys
	KeySystemVoltage   = "V"      // System voltage
	KeyBatteryVoltage  = "B.V"    // Battery voltage
	KeyPVInputVoltage1 = "P.V"    // PV input voltage panel 1
	KeyPVInputVoltage2 = "P.V(2)" // PV input voltage panel 2

	// Current keys
	KeySystemCurrent   = "A"      // System current (charging current)
	KeyBatteryCurrent  = "B.A"    // Battery current
	KeyPVInputCurrent1 = "P.A"    // PV input current panel 1
	KeyPVInputCurrent2 = "P.A(2)" // PV input current panel 2

	// Power keys
	KeySystemPower   = "W"      // System power (charging power)
	KeyPVInputPower1 = "P.W"    // PV input power panel 1
	KeyPVInputPower2 = "P.W(2)" // PV input power panel 2

	// State of Charge
	KeyStateOfCharge = "SoC" // State of charge percentage

	// Energy
	KeyEnergyYieldToday = "V.Ar"    // Energy yield today (Ah)
	KeyEnergyWhToday    = "V.Wh.ar" // Energy yield today (Wh)

	// Temperature
	KeyTemperature = "T" // Device temperature

	// Algorithm state
	KeyAlgorithmState = "Alg.S" // Charging algorithm state
	KeyStage          = "S"     // Stage

	// Error codes
	KeyErrorCode = "E" // Error code

	// Device identification
	KeyDeviceVersion  = "/Dev" // Device type string
	KeyDeviceRevision = "/Rev" // Firmware revision
	KeySerialNumber   = "/Sn"  // Serial number
)
