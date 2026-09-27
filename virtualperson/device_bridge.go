package virtualperson

import (
	"context"

	"github.com/funfunpayer/SamNPlayer/device"
)

// DeviceBridge is the Virtual Person view of Sam Neo 2 control.
// It mirrors device.Device so the plugin never talks BLE/Intiface directly.
// Production wiring: wrap the same device.Device instance the player uses.
type DeviceBridge interface {
	Connect(ctx context.Context) error
	Disconnect() error
	SetVibration(intensity float64) error
	SetSuction(intensity float64) error
	Stop() error
}

// DeviceBridgeFrom adapts a device.Device to DeviceBridge.
// Returns nil if dev is nil.
func DeviceBridgeFrom(dev device.Device) DeviceBridge {
	if dev == nil {
		return nil
	}
	return &deviceAdapter{dev: dev}
}

type deviceAdapter struct {
	dev device.Device
}

func (a *deviceAdapter) Connect(ctx context.Context) error    { return a.dev.Connect(ctx) }
func (a *deviceAdapter) Disconnect() error                    { return a.dev.Disconnect() }
func (a *deviceAdapter) SetVibration(intensity float64) error { return a.dev.SetVibration(intensity) }
func (a *deviceAdapter) SetSuction(intensity float64) error   { return a.dev.SetSuction(intensity) }
func (a *deviceAdapter) Stop() error                          { return a.dev.Stop() }

// NullDevice is a no-op bridge for tests and animate-only mode.
type NullDevice struct{}

func (NullDevice) Connect(context.Context) error { return nil }
func (NullDevice) Disconnect() error             { return nil }
func (NullDevice) SetVibration(float64) error    { return nil }
func (NullDevice) SetSuction(float64) error      { return nil }
func (NullDevice) Stop() error                   { return nil }
