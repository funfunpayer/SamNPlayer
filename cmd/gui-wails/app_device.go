package main

import (
	"context"
	"fmt"
	"time"

	"github.com/funfunpayer/SamNPlayer/device"
	"github.com/funfunpayer/SamNPlayer/logging"
)

// Dieser Tab existiert, weil bisher nirgends sichtbar war, ob überhaupt ein
// Gerät gefunden und richtig erkannt wurde: das Gerät wurde ausschließlich
// innerhalb von StartPlayback/StartTraining erzeugt, verbunden und danach
// wieder verworfen. Ein Verbindungsfehler fiel damit erst mitten in einer
// Wiedergabe auf, und es gab keine Möglichkeit, die Ansteuerung isoliert zu
// prüfen.
//
// Bewusste Einschränkung: BLE erlaubt ohnehin nur eine Verbindung zum Gerät,
// deshalb schließen sich Testverbindung und laufende Session gegenseitig aus.
// Das wird explizit gemeldet, statt im Hintergrund die eine Verbindung durch
// die andere zu ersetzen - eine stillschweigende Übernahme wäre bei einem
// Gerät, das gerade läuft, die schlechtere Überraschung.

// TestConnectionTimeout begrenzt den BLE-Scan im Geräte-Tab.
const testConnectTimeout = 20 * time.Second

// DeviceStatus ist der Zustand, den der Geräte-Tab anzeigt.
type DeviceStatus struct {
	Connected     bool   `json:"connected"`
	Mock          bool   `json:"mock"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	RSSI          int    `json:"rssi"`
	SessionActive bool   `json:"sessionActive"`
}

// GetDeviceStatus liefert den aktuellen Verbindungszustand der Testverbindung.
func (a *App) GetDeviceStatus() DeviceStatus {
	a.stateMu.RLock()
	dev := a.testDevice
	isMock := a.testDeviceMock
	session := a.sessionActive
	a.stateMu.RUnlock()

	st := DeviceStatus{SessionActive: session, Mock: isMock}
	if dev == nil {
		return st
	}
	st.Connected = true
	if intiface, ok := dev.(*device.Intiface); ok {
		info := intiface.Info()
		st.Connected = info.Connected
		st.Name = info.Name
		st.Address = info.Address
		return st
	}
	if real, ok := dev.(*device.SamNeo2); ok {
		info := real.Info()
		st.Connected = info.Connected
		st.Name = info.Name
		st.Address = info.Address
		st.RSSI = info.RSSI
	} else {
		st.Name = "Mock-Gerät (keine echte Hardware)"
	}
	return st
}

// ConnectDevice baut eine dauerhafte Testverbindung auf, unabhängig von
// Wiedergabe und Training.
// ConnectDevice verbindet mit dem Gerät. transport wählt den Weg:
// "ble" (Standard, eigener Bluetooth-Adapter), "intiface" (über einen
// laufenden Buttplug-Server, meist Intiface Central) oder "mock".
//
// Der Intiface-Weg braucht keinen eigenen Bluetooth-Adapter und funktioniert
// mit jedem von Buttplug unterstützten Gerät. Der direkte BLE-Weg bleibt
// daneben bestehen, weil er ohne Zusatzsoftware auskommt.
func (a *App) ConnectDeviceVia(transport, url string) (DeviceStatus, error) {
	a.stateMu.Lock()
	if a.sessionActive {
		a.stateMu.Unlock()
		return a.GetDeviceStatus(), fmt.Errorf("es läuft gerade eine Wiedergabe oder ein " +
			"Training - bitte zuerst beenden, das Gerät kann nur eine Verbindung " +
			"gleichzeitig halten")
	}
	if a.testDevice != nil {
		a.stateMu.Unlock()
		return a.GetDeviceStatus(), fmt.Errorf("es besteht bereits eine Verbindung - zuerst trennen")
	}
	a.stateMu.Unlock()

	var dev device.Device
	switch transport {
	case "mock":
		dev = device.NewMock(true)
	case "intiface":
		dev = device.NewIntiface(url)
	default:
		dev = device.NewSamNeo2(device.SamNeo2Protocol{})
	}

	// Intiface bekommt mehr Zeit: der Server sucht unter Umständen noch
	// selbst nach Geräten.
	timeout := testConnectTimeout
	if transport == "intiface" {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := dev.Connect(ctx); err != nil {
		logging.Warn("geraet: Verbindung fehlgeschlagen", "weg", transport, "fehler", err)
		return a.GetDeviceStatus(), err
	}

	a.stateMu.Lock()
	a.testDevice = dev
	a.testDeviceMock = transport == "mock"
	a.stateMu.Unlock()

	// Erfolgreiche Wahl merken - beim nächsten Start ist sie voreingestellt.
	_ = a.settings.Set(prefDeviceTransport, transport)
	if transport == "intiface" {
		_ = a.settings.Set(prefIntifaceURL, url)
	}

	st := a.GetDeviceStatus()
	logging.Info("geraet: verbunden", "weg", transport, "name", st.Name, "adresse", st.Address)
	return st, nil
}

func (a *App) ConnectDevice(mock bool) (DeviceStatus, error) {
	a.stateMu.Lock()
	if a.sessionActive {
		a.stateMu.Unlock()
		return a.GetDeviceStatus(), fmt.Errorf("es läuft gerade eine Wiedergabe oder ein Training - " +
			"bitte zuerst beenden, das Gerät kann nur eine Verbindung gleichzeitig halten")
	}
	if a.testDevice != nil {
		a.stateMu.Unlock()
		return a.GetDeviceStatus(), fmt.Errorf("es besteht bereits eine Verbindung - zuerst trennen")
	}
	a.stateMu.Unlock()

	var dev device.Device
	if mock {
		dev = device.NewMock(true)
	} else {
		dev = device.NewSamNeo2(device.SamNeo2Protocol{})
	}

	ctx, cancel := context.WithTimeout(context.Background(), testConnectTimeout)
	defer cancel()
	if err := dev.Connect(ctx); err != nil {
		logging.Warn("geraet: Verbindung fehlgeschlagen", "mock", mock, "fehler", err)
		return a.GetDeviceStatus(), err
	}

	a.stateMu.Lock()
	a.testDevice = dev
	a.testDeviceMock = mock
	a.stateMu.Unlock()

	st := a.GetDeviceStatus()
	logging.Info("geraet: verbunden", "mock", mock, "name", st.Name, "adresse", st.Address, "rssi", st.RSSI)
	return st, nil
}

// DisconnectDevice trennt die Testverbindung wieder.
func (a *App) DisconnectDevice() (DeviceStatus, error) {
	a.stateMu.Lock()
	dev := a.testDevice
	a.testDevice = nil
	a.testDeviceMock = false
	a.stateMu.Unlock()

	if dev == nil {
		return a.GetDeviceStatus(), nil
	}
	// Vor dem Trennen immer ausschalten - sonst läuft ein Gerät weiter, das
	// gerade noch einen Testimpuls bekommen hat.
	if err := dev.Stop(); err != nil {
		logging.Warn("geraet: Stop vor dem Trennen fehlgeschlagen", "fehler", err)
	}
	err := dev.Disconnect()
	logging.Info("geraet: getrennt", "fehler", err)
	return a.GetDeviceStatus(), err
}

// testDeviceOrErr liefert die aktive Testverbindung oder einen sprechenden Fehler.
func (a *App) testDeviceOrErr() (device.Device, error) {
	a.stateMu.RLock()
	dev := a.testDevice
	session := a.sessionActive
	a.stateMu.RUnlock()
	if session {
		return nil, fmt.Errorf("während einer laufenden Wiedergabe oder eines Trainings ist kein Gerätetest möglich")
	}
	if dev == nil {
		return nil, fmt.Errorf("kein Gerät verbunden - im Geräte-Tab zuerst verbinden")
	}
	return dev, nil
}

// TestVibration setzt die Vibration direkt (0.0-1.0). Das Gerät kennt
// intern 11 Stufen (0-10), siehe device/protocol.go.
func (a *App) TestVibration(intensity float64) error {
	dev, err := a.testDeviceOrErr()
	if err != nil {
		return err
	}
	logging.Info("geraet: Test Vibration", "intensitaet", intensity)
	return dev.SetVibration(clamp01(intensity))
}

// TestSuction setzt den Sog direkt (0.0-1.0). Das Gerät kennt intern 6
// Stufen (0-5) und ist dabei stufenlos ansteuerbar, keine festen Muster.
func (a *App) TestSuction(intensity float64) error {
	dev, err := a.testDeviceOrErr()
	if err != nil {
		return err
	}
	logging.Info("geraet: Test Sog", "intensitaet", intensity)
	return dev.SetSuction(clamp01(intensity))
}

// TestStop schaltet beide Kanäle sofort ab.
func (a *App) TestStop() error {
	dev, err := a.testDeviceOrErr()
	if err != nil {
		return err
	}
	logging.Info("geraet: Test Stop")
	return dev.Stop()
}

// rawCapable ist die Schnittstelle für den Rohwert-Test. Nur das echte
// BLE-Gerät und das Mock bieten sie an.
type rawCapable interface {
	SetVibrationRaw(byte) error
	SetSuctionRaw(byte) error
}

// TestRawValue sendet einen Stufenwert DIREKT, ohne die Quantisierung auf
// 0-10 (Vibration) bzw. 0-5 (Sog).
//
// Diese Bereiche stammen aus der Buttplug-Gerätekonfiguration
// (svakom-sam2.yml) - das ist die StepCount, auf die Buttplug quantisiert,
// und nicht notwendigerweise eine Grenze der Firmware. Das Byte im Paket
// kann 0-255 tragen. Ob das Gerät Zwischenwerte oder höhere Werte annimmt,
// lässt sich ausschließlich am echten Gerät feststellen - und genau dafür
// ist diese Funktion da.
//
// Die Antwort hat Folgen über das Training hinaus: ist die Auflösung feiner,
// ist auch die Funscript-Wiedergabe unnötig grob gerastert.
func (a *App) TestRawValue(channel string, value int) error {
	dev, err := a.testDeviceOrErr()
	if err != nil {
		return err
	}
	raw, ok := dev.(rawCapable)
	if !ok {
		return fmt.Errorf("dieses Gerät unterstützt keine Rohwerte")
	}
	if value < 0 || value > 255 {
		return fmt.Errorf("Rohwert muss zwischen 0 und 255 liegen (ist %d)", value)
	}
	logging.Info("geraet: Rohwert-Test", "kanal", channel, "wert", value)
	switch channel {
	case "vibration":
		return raw.SetVibrationRaw(byte(value))
	case "suction":
		return raw.SetSuctionRaw(byte(value))
	default:
		return fmt.Errorf("unbekannter Kanal %q", channel)
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
