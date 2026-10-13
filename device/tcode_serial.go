package device

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"go.bug.st/serial"
)

// TCode over USB serial (OSR2 / SR6 with USB firmware; D5). The common
// firmwares listen at 115200 baud, 8N1.

// TCodeSerialBaud is the baud rate of the OSR / SR6 USB firmwares.
const TCodeSerialBaud = 115200

// NewTCodeSerial creates a TCode device on a serial port ("COM3",
// "/dev/ttyUSB0", "/dev/cu.usbserial-…").
func NewTCodeSerial(port string) *TCode {
	port = strings.TrimSpace(port)
	return &TCode{address: port, serialPort: port, openSerial: openTCodeSerial}
}

func openTCodeSerial(port string) (io.WriteCloser, error) {
	if port == "" {
		return nil, fmt.Errorf("no serial port chosen")
	}
	return serial.Open(port, &serial.Mode{BaudRate: TCodeSerialBaud, DataBits: 8, Parity: serial.NoParity, StopBits: serial.OneStopBit})
}

// SerialPorts lists the serial ports of this computer, sorted.
func SerialPorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, err
	}
	sort.Strings(ports)
	return ports, nil
}
