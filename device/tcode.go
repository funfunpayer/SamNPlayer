package device

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TCode (docs/DEVICE_SUPPORT.md, D5): the open text protocol of OSR2 / SR6
// and other DIY strokers. A command is "<axis><value>[I<ms>]", the value a
// decimal fraction written as digits ("L05" = 0.5, "L09999" = 0.9999);
// several commands share one line, a newline executes them. "DSTOP" stops.
//
// This transport talks TCode over the network (the Wi-Fi firmware of the
// ESP32 boards listens on UDP or TCP). It only writes: TCode needs no
// replies for motion. Driven today: L0 (stroke) via SetPosition; the other
// axes come with multi-axis output (D4).

// tcodeDigits: value precision (4 digits = 0–9999).
const tcodeDigits = 4

// TCodeValue formats pos (0–1) as TCode digits, e.g. 0.5 → "5000".
func TCodeValue(pos float64) string {
	if !(pos >= 0) {
		pos = 0
	}
	if pos > 1 {
		pos = 1
	}
	// Digits are the decimal fraction ("8000" = 0.8); 1.0 caps at "9999".
	v := int(math.Round(pos * math.Pow10(tcodeDigits)))
	if top := int(math.Pow10(tcodeDigits)) - 1; v > top {
		v = top
	}
	return fmt.Sprintf("%0*d", tcodeDigits, v)
}

// TCodeMove builds one linear move: axis to pos within d ("L05000I250").
func TCodeMove(axis string, pos float64, d time.Duration) string {
	ms := d.Milliseconds()
	if ms < 1 {
		ms = 1
	}
	return fmt.Sprintf("%s%sI%d", axis, TCodeValue(pos), ms)
}

// TCode is a network TCode device (udp://host:port or tcp://host:port).
type TCode struct {
	address string // as given, for display

	mu        sync.Mutex
	conn      io.WriteCloser
	connected bool
	dial      func(ctx context.Context, network, addr string) (net.Conn, error)
}

// DefaultTCodePort is the port the common OSR / SR6 Wi-Fi firmwares use.
const DefaultTCodePort = "8000"

// NewTCode creates a TCode device for address ("udp://192.168.1.50:8000",
// "tcp://osr.local:8000"; without scheme UDP, without port 8000).
func NewTCode(address string) *TCode {
	var d net.Dialer
	return &TCode{address: strings.TrimSpace(address), dial: d.DialContext}
}

// ParseTCodeAddress splits address into network ("udp" / "tcp") and host:port.
func ParseTCodeAddress(address string) (network, hostport string, err error) {
	a := strings.TrimSpace(address)
	if a == "" {
		return "", "", fmt.Errorf("tcode: no address — enter the device's IP, e.g. 192.168.1.50")
	}
	network = "udp"
	if strings.Contains(a, "://") {
		u, perr := url.Parse(a)
		if perr != nil || u.Host == "" {
			return "", "", fmt.Errorf("tcode: invalid address %q", address)
		}
		switch strings.ToLower(u.Scheme) {
		case "udp", "tcp":
			network = strings.ToLower(u.Scheme)
		default:
			return "", "", fmt.Errorf("tcode: use udp:// or tcp:// (got %q)", u.Scheme)
		}
		a = u.Host
	}
	if _, _, splitErr := net.SplitHostPort(a); splitErr != nil {
		a = net.JoinHostPort(strings.Trim(a, "[]"), DefaultTCodePort)
	}
	return network, a, nil
}

// Connect opens the socket. UDP has no handshake, so a wrong address only
// shows when the device does not move; TCP fails here.
func (t *TCode) Connect(ctx context.Context) error {
	network, hostport, err := ParseTCodeAddress(t.address)
	if err != nil {
		return err
	}
	conn, err := t.dial(ctx, network, hostport)
	if err != nil {
		return fmt.Errorf("tcode: no connection to %s (%s): %w", hostport, network, err)
	}
	t.mu.Lock()
	t.conn = conn
	t.connected = true
	t.mu.Unlock()
	return nil
}

// Disconnect stops the device and closes the socket.
func (t *TCode) Disconnect() error {
	_ = t.Stop()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil
	}
	err := t.conn.Close()
	t.conn = nil
	t.connected = false
	return err
}

func (t *TCode) writeLine(line string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return fmt.Errorf("tcode: not connected")
	}
	if _, err := io.WriteString(t.conn, line+"\n"); err != nil {
		t.connected = false
		return fmt.Errorf("tcode: send failed: %w", err)
	}
	return nil
}

// SetVibration / SetSuction: an OSR2 / SR6 has neither — no-ops, so the
// player's feel channels do not error on a stroker.
func (t *TCode) SetVibration(float64) error { return nil }
func (t *TCode) SetSuction(float64) error   { return nil }

// Stop halts every axis immediately.
func (t *TCode) Stop() error {
	t.mu.Lock()
	open := t.conn != nil
	t.mu.Unlock()
	if !open {
		return nil
	}
	return t.writeLine("DSTOP")
}

// HasLinear: a TCode device always has L0.
func (t *TCode) HasLinear() bool { return true }

// SetPosition moves L0 (stroke) to pos within d.
func (t *TCode) SetPosition(pos float64, d time.Duration) error {
	return t.writeLine(TCodeMove("L0", pos, d))
}

// DeviceCapabilities: stroke (L0) is driven; the extra axes of an SR6 are
// listed by D4 output later, not guessed here.
func (t *TCode) DeviceCapabilities() Capabilities {
	return Capabilities{
		Device:   "TCode device",
		Features: []Feature{{Channel: ChStroke, Kind: KindLinear, Index: 0, Actuator: "L0", Driven: true}},
	}
}

// Info reports the connection for the GUI.
func (t *TCode) Info() ConnectionInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	network, hostport, err := ParseTCodeAddress(t.address)
	addr := t.address
	if err == nil {
		addr = network + "://" + hostport
	}
	return ConnectionInfo{Connected: t.connected, Name: "TCode device (Stroke L0)", Address: addr}
}
