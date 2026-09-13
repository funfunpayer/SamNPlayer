package device

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Ein nachgebauter Buttplug-Server. Er antwortet nach der offenen
// Protokollbeschreibung und zeichnet auf, was der Client sendet - damit ist
// prüfbar, dass wir uns an das Protokoll halten, ohne einen echten Server
// und ein echtes Gerät zu brauchen.
type fakeButtplug struct {
	mu       sync.Mutex
	received []map[string]json.RawMessage
	// deviceMessages bestimmt, welche Kanäle das gemeldete Gerät kann.
	deviceMessages map[string]any
	noDevices      bool
}

func (f *fakeButtplug) record(message map[string]json.RawMessage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.received = append(f.received, message)
}

func (f *fakeButtplug) names() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, message := range f.received {
		for name := range message {
			out = append(out, name)
		}
	}
	return out
}

func (f *fakeButtplug) scalarCommands() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []map[string]any
	for _, message := range f.received {
		raw, ok := message["ScalarCmd"]
		if !ok {
			continue
		}
		var body map[string]any
		if json.Unmarshal(raw, &body) == nil {
			out = append(out, body)
		}
	}
	return out
}

func (f *fakeButtplug) handler() http.HandlerFunc {
	upgrader := websocket.Upgrader{}
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			var batch []map[string]json.RawMessage
			if err := conn.ReadJSON(&batch); err != nil {
				return
			}
			for _, message := range batch {
				f.record(message)
				for name, raw := range message {
					var body map[string]any
					_ = json.Unmarshal(raw, &body)
					id := body["Id"]
					switch name {
					case "RequestServerInfo":
						_ = conn.WriteJSON([]any{map[string]any{"ServerInfo": map[string]any{
							"Id": id, "ServerName": "FakeServer",
							"MessageVersion": 3, "MaxPingTime": 5000,
						}}})
					case "RequestDeviceList":
						devices := []any{}
						if !f.noDevices {
							devices = append(devices, map[string]any{
								"DeviceIndex": 7, "DeviceName": "Testgerät",
								"DeviceMessages": f.deviceMessages,
							})
						}
						_ = conn.WriteJSON([]any{map[string]any{"DeviceList": map[string]any{
							"Id": id, "Devices": devices,
						}}})
					case "StartScanning", "StopScanning", "Ping", "ScalarCmd", "StopDeviceCmd":
						_ = conn.WriteJSON([]any{map[string]any{"Ok": map[string]any{"Id": id}}})
					}
				}
			}
		}
	}
}

func standardDeviceMessages() map[string]any {
	return map[string]any{
		"ScalarCmd": []any{
			map[string]any{"StepCount": 10, "ActuatorType": "Vibrate", "FeatureDescriptor": "Vib"},
			map[string]any{"StepCount": 5, "ActuatorType": "Constrict", "FeatureDescriptor": "Sog"},
		},
		"StopDeviceCmd": map[string]any{},
	}
}

func startFake(t *testing.T, fake *fakeButtplug) (string, func()) {
	t.Helper()
	server := httptest.NewServer(fake.handler())
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	return url, server.Close
}

// Der Kern: Anmeldung, Geräteübernahme und Ansteuerung beider Kanäle.
func TestIntifaceConnectAndControl(t *testing.T) {
	fake := &fakeButtplug{deviceMessages: standardDeviceMessages()}
	url, stop := startFake(t, fake)
	defer stop()

	dev := NewIntiface(url)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dev.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer dev.Disconnect()

	info := dev.Info()
	if !info.Connected {
		t.Error("nach Connect muss die Verbindung als bestehend gelten")
	}
	if !strings.Contains(info.Name, "Testgerät") {
		t.Errorf("Gerätename fehlt in der Anzeige: %q", info.Name)
	}
	// Die Anzeige soll sagen, WELCHE Kanäle gefunden wurden - sonst merkt
	// man erst beim Abspielen, dass der Sog fehlt.
	if !strings.Contains(info.Name, "Vibration") || !strings.Contains(info.Name, "Sog") {
		t.Errorf("gefundene Kanäle fehlen in der Anzeige: %q", info.Name)
	}

	names := fake.names()
	if !contains(names, "RequestServerInfo") || !contains(names, "RequestDeviceList") {
		t.Errorf("Anmeldefolge unvollständig: %v", names)
	}

	if err := dev.SetVibration(0.5); err != nil {
		t.Fatalf("SetVibration: %v", err)
	}
	if err := dev.SetSuction(1.0); err != nil {
		t.Fatalf("SetSuction: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	commands := fake.scalarCommands()
	if len(commands) < 2 {
		t.Fatalf("erwartet 2 ScalarCmd, bekam %d", len(commands))
	}
	for _, cmd := range commands {
		if idx, _ := cmd["DeviceIndex"].(float64); int(idx) != 7 {
			t.Errorf("falscher DeviceIndex: %v", cmd["DeviceIndex"])
		}
	}
}

// Werte müssen auf 0-1 begrenzt werden: das Protokoll verlangt es, und ein
// Wert über 1 würde vom Server abgelehnt - die Wiedergabe bräche dann
// mitten im Ablauf ab.
func TestIntifaceClampsValues(t *testing.T) {
	fake := &fakeButtplug{deviceMessages: standardDeviceMessages()}
	url, stop := startFake(t, fake)
	defer stop()

	dev := NewIntiface(url)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dev.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer dev.Disconnect()

	_ = dev.SetVibration(3.5)
	_ = dev.SetVibration(-2.0)
	time.Sleep(100 * time.Millisecond)

	for _, cmd := range fake.scalarCommands() {
		scalars, _ := cmd["Scalars"].([]any)
		for _, entry := range scalars {
			value, _ := entry.(map[string]any)["Scalar"].(float64)
			if value < 0 || value > 1 {
				t.Errorf("Wert außerhalb 0-1 gesendet: %v", value)
			}
		}
	}
}

// Ohne Gerät muss die Verbindung mit einer verständlichen Meldung scheitern,
// statt scheinbar zu gelingen und beim ersten Befehl still nichts zu tun.
func TestIntifaceWithoutDeviceFails(t *testing.T) {
	fake := &fakeButtplug{noDevices: true, deviceMessages: standardDeviceMessages()}
	url, stop := startFake(t, fake)
	defer stop()

	dev := NewIntiface(url)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := dev.Connect(ctx)
	if err == nil {
		dev.Disconnect()
		t.Fatal("ohne Gerät darf Connect nicht gelingen")
	}
	if !strings.Contains(err.Error(), "Gerät") {
		t.Errorf("Meldung nennt das Problem nicht: %v", err)
	}
}

// Ein Gerät, das nur Vibration kann, soll trotzdem nutzbar sein - nur eben
// ohne Sog. Alles andere würde die meisten Buttplug-Geräte ausschließen.
func TestIntifaceVibrationOnlyDevice(t *testing.T) {
	fake := &fakeButtplug{deviceMessages: map[string]any{
		"ScalarCmd": []any{
			map[string]any{"StepCount": 20, "ActuatorType": "Vibrate"},
		},
	}}
	url, stop := startFake(t, fake)
	defer stop()

	dev := NewIntiface(url)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dev.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer dev.Disconnect()

	if err := dev.SetSuction(0.8); err != nil {
		t.Errorf("fehlender Sog-Kanal darf keinen Fehler ergeben: %v", err)
	}
	if err := dev.SetVibration(0.4); err != nil {
		t.Errorf("SetVibration: %v", err)
	}
	info := dev.Info()
	if strings.Contains(info.Name, "Sog") {
		t.Errorf("Anzeige behauptet einen Kanal, den es nicht gibt: %q", info.Name)
	}
}

// Ist kein Server da, muss die Meldung auf Intiface Central hinweisen -
// das ist die mit Abstand häufigste Ursache.
func TestIntifaceNoServer(t *testing.T) {
	dev := NewIntiface("ws://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := dev.Connect(ctx)
	if err == nil {
		t.Fatal("ohne Server darf Connect nicht gelingen")
	}
	if !strings.Contains(err.Error(), "Intiface") {
		t.Errorf("Meldung hilft nicht weiter: %v", err)
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// Der Hauptanwendungsfall ist: Intiface Central läuft auf dem Handy, der
// Player auf dem Rechner. Dann tippt man eine IP-Adresse ein, keine
// vollständige ws://-URL. Ohne Umformung scheitert die Verbindung mit einer
// Meldung über ein unbekanntes Schema, die niemandem weiterhilft.
func TestNormalizeIntifaceURL(t *testing.T) {
	cases := map[string]string{
		"":                        "ws://127.0.0.1:12345",
		"   ":                     "ws://127.0.0.1:12345",
		"192.168.1.50":            "ws://192.168.1.50:12345",
		"192.168.1.50:12345":      "ws://192.168.1.50:12345",
		"192.168.1.50:9999":       "ws://192.168.1.50:9999",
		"ws://192.168.1.50:12345": "ws://192.168.1.50:12345",
		"ws://192.168.1.50":       "ws://192.168.1.50:12345",
		"ws://10.0.0.5:12345/":    "ws://10.0.0.5:12345",
		"handy.local":             "ws://handy.local:12345",
	}
	for input, want := range cases {
		if got := NormalizeIntifaceURL(input); got != want {
			t.Errorf("NormalizeIntifaceURL(%q) = %q, erwartet %q", input, got, want)
		}
	}
}

// Die Fehlermeldung muss die wahrscheinlichste Ursache nennen - und die
// unterscheidet sich deutlich, je nachdem ob der Server lokal oder im
// Netzwerk laufen soll.
func TestIntifaceHintDistinguishesLocalAndNetwork(t *testing.T) {
	refused := errorString("dial tcp: connection refused")

	local := intifaceHint("ws://127.0.0.1:12345", refused)
	if !strings.Contains(local, "Server starten") {
		t.Errorf("lokaler Hinweis unbrauchbar: %q", local)
	}
	if strings.Contains(local, "WLAN") {
		t.Errorf("lokaler Hinweis spricht fälschlich vom Netzwerk: %q", local)
	}

	remote := intifaceHint("ws://192.168.1.50:12345", refused)
	if !strings.Contains(remote, "Netzwerk") {
		t.Errorf("Netzwerk-Hinweis fehlt: %q", remote)
	}

	timeout := intifaceHint("ws://192.168.1.50:12345",
		errorString("dial tcp: i/o timeout"))
	if !strings.Contains(timeout, "WLAN") {
		t.Errorf("bei Zeitüberschreitung im Netzwerk fehlt der WLAN-Hinweis: %q", timeout)
	}

	host := intifaceHint("ws://handy.local:12345", errorString("no such host"))
	if !strings.Contains(host, "IP-Adresse") {
		t.Errorf("bei unauflösbarem Namen fehlt der Hinweis auf die IP: %q", host)
	}
}

type errorString string

func (e errorString) Error() string { return string(e) }
