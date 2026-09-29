package player

import (
	"context"
	"testing"
	"time"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func lastOut(d *recordingDev) (float64, float64) {
	vib, suc := d.snapshot()
	if len(vib) == 0 || len(suc) == 0 {
		return -1, -1
	}
	return vib[len(vib)-1], suc[len(suc)-1]
}

// Video pausiert (native Steuerung/Leertaste): das Frontend schickt keine
// Positionen mehr. Vorher blieb das Gerät auf dem letzten Wert stehen, bis
// weitergespielt oder gestoppt wurde. Jetzt geht es nach syncStaleAfter
// auf 0 und folgt bei der nächsten Position wieder dem Skript.
func TestSyncGoesQuietWhenPositionsStop(t *testing.T) {
	old := syncStaleAfter
	syncStaleAfter = 150 * time.Millisecond
	defer func() { syncStaleAfter = old }()

	dev := &recordingDev{}
	p := New(dev)
	frames := []funscript.Frame{{At: 0, Vibration: 0.8, Suction: 0.6}, {At: 10000, Vibration: 0.8, Suction: 0.6}}
	positions := make(chan int64, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Sync(ctx, frames, positions) }()

	positions <- 1000
	time.Sleep(50 * time.Millisecond)
	if v, s := lastOut(dev); v != 0.8 || s != 0.6 {
		t.Fatalf("läuft nicht: vib %v suc %v", v, s)
	}
	time.Sleep(300 * time.Millisecond) // "Video pausiert"
	if v, s := lastOut(dev); v != 0 || s != 0 {
		t.Errorf("ohne Positionen muss das Gerät auf 0: vib %v suc %v", v, s)
	}
	positions <- 1300 // weiter
	time.Sleep(50 * time.Millisecond)
	if v, s := lastOut(dev); v != 0.8 || s != 0.6 {
		t.Errorf("nach neuer Position nicht zurück: vib %v suc %v", v, s)
	}
	close(positions)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
