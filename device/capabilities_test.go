package device

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestButtplugFeaturesNeo2Like(t *testing.T) {
	f := buttplugFeatures(standardDeviceMessages(), 0, 1)
	if len(f) != 2 || f[0].Channel != ChVibration || !f[0].Driven || f[1].Channel != ChSuction || !f[1].Driven {
		t.Fatalf("features = %+v", f)
	}
}

func TestButtplugFeaturesMultiAxis(t *testing.T) {
	msgs := map[string]any{
		"ScalarCmd": []any{
			map[string]any{"ActuatorType": "Vibrate"},
			map[string]any{"ActuatorType": "Vibrate"},
			map[string]any{"ActuatorType": "Rotate"},
			map[string]any{"ActuatorType": "Unknown"},
		},
		"LinearCmd": []any{map[string]any{"StepCount": 100}},
		"RotateCmd": []any{map[string]any{"StepCount": 20}},
	}
	f := buttplugFeatures(msgs, 0, -1)
	got := []string{}
	for _, x := range f {
		got = append(got, x.Channel+":"+x.Kind)
	}
	want := "vibration:scalar vibration2:scalar rotate:scalar stroke:linear rotate2:rotate"
	if strings.Join(got, " ") != want {
		t.Fatalf("channels = %q, want %q", strings.Join(got, " "), want)
	}
	if !f[0].Driven || f[1].Driven || f[3].Driven {
		t.Fatalf("only the first vibration is driven today: %+v", f)
	}
}

// The Handy over Intiface lists LinearCmd only. It used to be skipped
// ("no usable channels"); now it connects and reports its stroke channel.
func TestIntifaceAdoptsStroker(t *testing.T) {
	fake := &fakeButtplug{deviceMessages: map[string]any{
		"LinearCmd":     []any{map[string]any{"StepCount": 100, "ActuatorType": "Position"}},
		"StopDeviceCmd": map[string]any{},
	}}
	url, server := startFake(t, fake)
	defer server.Close()
	dev := NewIntiface(url)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dev.Connect(ctx); err != nil {
		t.Fatalf("a stroker must connect: %v", err)
	}
	defer dev.Disconnect()
	caps := CapabilitiesOf(dev)
	if !caps.Has(ChStroke) || caps.Has(ChVibration) || caps.Features[0].Kind != KindLinear || caps.Features[0].Driven {
		t.Fatalf("caps = %+v", caps)
	}
	if err := dev.SetVibration(0.5); err != nil {
		t.Fatalf("no vibration motor must not error: %v", err)
	}
	if name := dev.Info().Name; !strings.Contains(name, "Stroke (not driven yet)") {
		t.Fatalf("info names the channel honestly: %q", name)
	}
}

func TestCapabilitiesOfNeo2AndMock(t *testing.T) {
	m := CapabilitiesOf(NewMock(false))
	if got := strings.Join(m.Channels(), ","); got != "vibration,suction" {
		t.Fatalf("mock channels %q", got)
	}
	for _, f := range m.Features {
		if !f.Driven {
			t.Fatalf("mock channels are driven: %+v", f)
		}
	}
	// A device without its own report falls back to the two classic channels.
	if got := strings.Join(CapabilitiesOf(bareDevice{}).Channels(), ","); got != "vibration,suction" {
		t.Fatalf("fallback channels %q", got)
	}
}

// bareDevice reports no capabilities of its own.
type bareDevice struct{}

func (bareDevice) Connect(context.Context) error { return nil }
func (bareDevice) Disconnect() error             { return nil }
func (bareDevice) SetVibration(float64) error    { return nil }
func (bareDevice) SetSuction(float64) error      { return nil }
func (bareDevice) Stop() error                   { return nil }
