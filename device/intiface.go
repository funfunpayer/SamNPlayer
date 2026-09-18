package device

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/funfunpayer/SamNPlayer/logging"
)

// Intiface spricht mit einem Buttplug-Server (Intiface Central) über
// WebSocket, statt selbst eine Bluetooth-Verbindung aufzubauen.
//
// Warum das den Aufwand lohnt:
//
//   - Kein eigener Bluetooth-Adapter nötig. Intiface Central kümmert sich um
//     Verbindung, Pairing und Wiederverbindung - das sind erfahrungsgemäß die
//     fehleranfälligsten Teile, und sie sind dort für dutzende Geräte
//     erprobt.
//   - Es funktioniert mit JEDEM von Buttplug unterstützten Gerät, nicht nur
//     mit dem Sam Neo 2. Der eigene BLE-Pfad bleibt daneben bestehen, weil er
//     ohne Zusatzsoftware auskommt.
//
// Umgesetzt ist der Buttplug-Nachrichtenaustausch nach der offenen
// Protokollbeschreibung, nicht durch Übernahme fremden Codes. Nachrichten
// sind JSON-Objekte, jede in ein Array verpackt, mit fortlaufender Id.
//
// Bewusst nicht umgesetzt: Geräteauswahl per Index, Linear- und
// Rotationsbefehle. Sensoren nur für den Akku (BatteryLevelCmd), wenn das
// Gerät ihn meldet. Wir brauchen genau zwei Steuerkanäle auf dem ersten
// gefundenen Gerät - alles andere wäre Vorrat ohne Nutzung.
type Intiface struct {
	url string

	mu        sync.Mutex
	conn      *websocket.Conn
	nextID    int
	deviceIdx int
	// scalarIndex bildet unsere Kanäle auf die ScalarCmd-Indizes des
	// Geräts ab. Welcher Index welche Funktion hat, sagt erst die
	// Gerätebeschreibung - fest verdrahtete Annahmen wären hier falsch.
	vibrateIdx   int
	constrictIdx int
	deviceName   string
	connected    bool
	hasBattery   bool
	batteryPct   int
	batteryOK    bool
	batteryAt    time.Time

	stopPing chan struct{}
}

const (
	intifaceDefaultURL = "ws://127.0.0.1:12345"
	intifaceTimeout    = 8 * time.Second
	// Buttplug-Server erwarten regelmäßige Ping-Nachrichten, sonst stoppen
	// sie aus Sicherheitsgründen alle Geräte. Der Server nennt in ServerInfo
	// ein Höchstintervall; wir senden deutlich häufiger.
	intifacePingInterval = 2 * time.Second
)

// NormalizeIntifaceURL macht aus einer Nutzereingabe eine gültige
// WebSocket-Adresse.
//
// Der Grund ist der Hauptanwendungsfall: Intiface Central läuft auf dem
// Handy, der Player auf dem Rechner. Dann tippt man eine IP-Adresse ein,
// keine vollständige ws://-URL - und ohne diese Umformung scheitert die
// Verbindung mit einer Meldung über ein unbekanntes Schema, was niemandem
// weiterhilft.
//
// Akzeptiert wird alles davon:
//
//	(leer)                  -> ws://127.0.0.1:12345
//	192.168.1.50            -> ws://192.168.1.50:12345
//	192.168.1.50:12345      -> ws://192.168.1.50:12345
//	ws://192.168.1.50:12345 -> unverändert
func NormalizeIntifaceURL(input string) string {
	value := strings.TrimSpace(input)
	if value == "" {
		return intifaceDefaultURL
	}
	if !strings.Contains(value, "://") {
		value = "ws://" + value
	}
	value = strings.TrimSuffix(value, "/")
	// Port ergänzen, wenn keiner angegeben wurde. Die Prüfung sucht nach
	// einem Doppelpunkt NACH dem Schema - der im "ws://" zählt nicht.
	rest := value[strings.Index(value, "://")+3:]
	if !strings.Contains(rest, ":") {
		value += ":12345"
	}
	return value
}

// NewIntiface erzeugt einen Client. Leere URL = Standardadresse von
// Intiface Central auf demselben Rechner.
func NewIntiface(url string) *Intiface {
	return &Intiface{url: NormalizeIntifaceURL(url), vibrateIdx: -1, constrictIdx: -1}
}

func (i *Intiface) send(message map[string]any) (int, error) {
	i.nextID++
	id := i.nextID
	for _, payload := range message {
		if m, ok := payload.(map[string]any); ok {
			m["Id"] = id
		}
	}
	return id, i.conn.WriteJSON([]any{message})
}

// readUntil liest Nachrichten, bis eine vom gesuchten Typ kommt. Andere
// Nachrichten (etwa DeviceAdded) werden dabei mitverarbeitet - der Server
// sendet sie unaufgefordert, und sie einfach zu verwerfen hieße, das
// angeschlossene Gerät zu übersehen.
func (i *Intiface) readUntil(want string, deadline time.Time) (map[string]any, error) {
	for time.Now().Before(deadline) {
		_ = i.conn.SetReadDeadline(deadline)
		var batch []map[string]json.RawMessage
		if err := i.conn.ReadJSON(&batch); err != nil {
			return nil, fmt.Errorf("intiface: Antwort nicht lesbar: %w", err)
		}
		for _, message := range batch {
			for name, raw := range message {
				var body map[string]any
				_ = json.Unmarshal(raw, &body)
				if name == "DeviceAdded" {
					i.adoptDevice(body)
				}
				if name == "Error" {
					return nil, fmt.Errorf("intiface: Server meldet Fehler: %v", body["ErrorMessage"])
				}
				if name == want {
					return body, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("intiface: keine Antwort vom Typ %q innerhalb der Wartezeit", want)
}

// adoptDevice übernimmt das erste Gerät, das die benötigten Kanäle bietet.
func (i *Intiface) adoptDevice(body map[string]any) {
	if i.deviceName != "" {
		return // schon eines übernommen
	}
	index, ok := body["DeviceIndex"].(float64)
	if !ok {
		return
	}
	name, _ := body["DeviceName"].(string)

	messages, _ := body["DeviceMessages"].(map[string]any)
	scalars, _ := messages["ScalarCmd"].([]any)
	for idx, entry := range scalars {
		feature, _ := entry.(map[string]any)
		actuator, _ := feature["ActuatorType"].(string)
		switch actuator {
		case "Vibrate":
			if i.vibrateIdx < 0 {
				i.vibrateIdx = idx
			}
		case "Constrict", "Oscillate", "Inflate":
			// Sog heißt je nach Gerät anders. Oscillate und Inflate sind
			// die naheliegenden Entsprechungen, wenn Constrict fehlt -
			// besser ein sinnvoll belegter zweiter Kanal als gar keiner.
			if i.constrictIdx < 0 {
				i.constrictIdx = idx
			}
		}
	}
	if i.vibrateIdx < 0 && i.constrictIdx < 0 {
		logging.Warn("intiface: Gerät ohne nutzbare Kanäle übersprungen", "geraet", name)
		return
	}
	// Akku nur anbieten, wenn Buttplug BatteryLevelCmd listet.
	if _, ok := messages["BatteryLevelCmd"]; ok {
		i.hasBattery = true
	}
	i.deviceIdx = int(index)
	i.deviceName = name
	logging.Info("intiface: Gerät übernommen", "geraet", name, "index", i.deviceIdx,
		"vibration", i.vibrateIdx, "sog", i.constrictIdx, "akku", i.hasBattery)
}

func (i *Intiface) Connect(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	dialer := websocket.Dialer{HandshakeTimeout: intifaceTimeout}
	conn, _, err := dialer.DialContext(ctx, i.url, nil)
	if err != nil {
		return fmt.Errorf("intiface: keine Verbindung zu %s - %s (%w)",
			i.url, intifaceHint(i.url, err), err)
	}
	i.conn = conn
	i.nextID = 0
	i.deviceName = ""
	i.vibrateIdx, i.constrictIdx = -1, -1
	i.hasBattery = false
	i.batteryOK = false
	i.batteryPct = 0
	i.batteryAt = time.Time{}

	deadline := time.Now().Add(intifaceTimeout)
	if _, err := i.send(map[string]any{"RequestServerInfo": map[string]any{
		"ClientName": "SamNPlayer", "MessageVersion": 3,
	}}); err != nil {
		conn.Close()
		return fmt.Errorf("intiface: Anmeldung fehlgeschlagen: %w", err)
	}
	info, err := i.readUntil("ServerInfo", deadline)
	if err != nil {
		conn.Close()
		return err
	}
	logging.Info("intiface: verbunden", "server", info["ServerName"])

	// Bereits verbundene Geräte abfragen. Ohne das würde nur ein Gerät
	// gefunden, das WÄHREND unserer Sitzung neu dazukommt.
	if _, err := i.send(map[string]any{"RequestDeviceList": map[string]any{}}); err != nil {
		conn.Close()
		return err
	}
	list, err := i.readUntil("DeviceList", deadline)
	if err == nil {
		if devices, ok := list["Devices"].([]any); ok {
			for _, entry := range devices {
				if body, ok := entry.(map[string]any); ok {
					i.adoptDevice(body)
				}
			}
		}
	}

	if i.deviceName == "" {
		// Kurz auf ein Gerät warten, das gerade erst gekoppelt wird.
		if _, err := i.send(map[string]any{"StartScanning": map[string]any{}}); err == nil {
			_, _ = i.readUntil("DeviceAdded", time.Now().Add(intifaceTimeout))
			_, _ = i.send(map[string]any{"StopScanning": map[string]any{}})
		}
	}
	if i.deviceName == "" {
		conn.Close()
		return fmt.Errorf("intiface: kein nutzbares Gerät gefunden - in Intiface Central " +
			"das Gerät verbinden und dann erneut versuchen")
	}

	i.connected = true
	i.stopPing = make(chan struct{})
	go i.pingLoop(i.stopPing)
	_ = i.conn.SetReadDeadline(time.Time{})
	return nil
}

// pingLoop hält die Verbindung am Leben. Bleibt der Ping aus, stoppt der
// Server aus Sicherheitsgründen alle Geräte - das ist gewollt, wäre hier
// aber ein Fehler.
func (i *Intiface) pingLoop(stop chan struct{}) {
	ticker := time.NewTicker(intifacePingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			i.mu.Lock()
			if i.conn != nil {
				_, _ = i.send(map[string]any{"Ping": map[string]any{}})
			}
			i.mu.Unlock()
		}
	}
}

func (i *Intiface) scalar(index int, value float64) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if !i.connected || i.conn == nil {
		return fmt.Errorf("intiface: nicht verbunden")
	}
	if index < 0 {
		return nil // Kanal auf diesem Gerät nicht vorhanden
	}
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	actuator := "Vibrate"
	if index == i.constrictIdx {
		actuator = "Constrict"
	}
	_, err := i.send(map[string]any{"ScalarCmd": map[string]any{
		"DeviceIndex": i.deviceIdx,
		"Scalars": []any{map[string]any{
			"Index": index, "Scalar": value, "ActuatorType": actuator,
		}},
	}})
	return err
}

func (i *Intiface) SetVibration(intensity float64) error {
	return i.scalar(i.vibrateIdx, intensity)
}

func (i *Intiface) SetSuction(intensity float64) error {
	return i.scalar(i.constrictIdx, intensity)
}

func (i *Intiface) Stop() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if !i.connected || i.conn == nil {
		return nil
	}
	_, err := i.send(map[string]any{"StopDeviceCmd": map[string]any{
		"DeviceIndex": i.deviceIdx,
	}})
	return err
}

func (i *Intiface) Disconnect() error {
	i.mu.Lock()
	stop := i.stopPing
	i.stopPing = nil
	conn := i.conn
	i.conn = nil
	i.connected = false
	i.mu.Unlock()

	if stop != nil {
		close(stop)
	}
	if conn == nil {
		return nil
	}
	// Vor dem Trennen abschalten - sonst läuft das Gerät weiter, und der
	// Server merkt den Abbruch erst über den ausbleibenden Ping.
	i.nextID++
	_ = conn.WriteJSON([]any{map[string]any{"StopDeviceCmd": map[string]any{
		"DeviceIndex": i.deviceIdx, "Id": i.nextID,
	}}})
	return conn.Close()
}

// Info liefert den Verbindungszustand für die Anzeige.
func (i *Intiface) Info() ConnectionInfo {
	i.mu.Lock()
	defer i.mu.Unlock()
	name := i.deviceName
	if name != "" {
		channels := []string{}
		if i.vibrateIdx >= 0 {
			channels = append(channels, "Vibration")
		}
		if i.constrictIdx >= 0 {
			channels = append(channels, "Sog")
		}
		name = fmt.Sprintf("%s (über Intiface, %s)", name, strings.Join(channels, " + "))
	}
	return ConnectionInfo{
		Connected:  i.connected,
		Name:       name,
		Address:    i.url,
		BatteryPct: i.batteryPct,
		BatteryOK:  i.batteryOK,
	}
}

// BatteryLevel fragt Buttplug BatteryLevelCmd ab, sofern das Gerät sie anbietet.
func (i *Intiface) BatteryLevel() (int, bool) {
	i.mu.Lock()
	if !i.connected || i.conn == nil || !i.hasBattery {
		i.mu.Unlock()
		return 0, false
	}
	if i.batteryOK && time.Since(i.batteryAt) < batteryCacheTTL {
		pct := i.batteryPct
		i.mu.Unlock()
		return pct, true
	}
	deviceIdx := i.deviceIdx
	if _, err := i.send(map[string]any{"BatteryLevelCmd": map[string]any{
		"DeviceIndex": deviceIdx,
	}}); err != nil {
		i.mu.Unlock()
		return 0, false
	}
	i.mu.Unlock()

	body, err := i.readUntil("BatteryLevelReading", time.Now().Add(3*time.Second))
	if err != nil {
		logging.Debug("intiface: Akku nicht lesbar", "fehler", err)
		return 0, false
	}
	raw, ok := body["BatteryLevel"].(float64)
	if !ok {
		return 0, false
	}
	pct := int(raw*100 + 0.5)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	i.mu.Lock()
	i.batteryPct = pct
	i.batteryOK = true
	i.batteryAt = time.Now()
	i.mu.Unlock()
	logging.Info("intiface: Akku gelesen", "prozent", pct)
	return pct, true
}

// intifaceHint nennt die wahrscheinlichste Ursache. Eine rohe
// Netzwerkfehlermeldung sagt dem Anwender nichts; die Ursachen
// unterscheiden sich außerdem deutlich, je nachdem ob der Server auf
// demselben Rechner oder im Netzwerk laufen soll.
func intifaceHint(url string, err error) string {
	local := strings.Contains(url, "127.0.0.1") || strings.Contains(url, "localhost")
	text := strings.ToLower(err.Error())

	switch {
	case strings.Contains(text, "refused"):
		if local {
			return "der Rechner ist erreichbar, aber auf diesem Port läuft kein Server. " +
				"In Intiface Central den Server starten"
		}
		return "das Gerät ist erreichbar, aber auf diesem Port läuft kein Server. " +
			"In Intiface Central den Server starten und prüfen, ob er für das Netzwerk " +
			"freigegeben ist (nicht nur für das Gerät selbst)"
	case strings.Contains(text, "timeout"), strings.Contains(text, "deadline"),
		strings.Contains(text, "no route"), strings.Contains(text, "unreachable"):
		if local {
			return "keine Antwort - läuft Intiface Central?"
		}
		return "keine Antwort. Sind Handy und Rechner im selben WLAN? " +
			"Stimmt die IP-Adresse? Blockiert eine Firewall den Port?"
	case strings.Contains(text, "no such host"):
		return "der Name ist nicht auflösbar - besser die IP-Adresse direkt eintragen"
	default:
		if local {
			return "läuft Intiface Central und ist der Server gestartet?"
		}
		return "Adresse und Netzwerkverbindung prüfen"
	}
}
