// Command SamNPlayer spielt eine .funscript-Datei auf einem
// SVAKOM Sam Neo 2 / Sam Neo 2 Pro (oder, mit --mock, nur in der Konsole) ab.
//
// Protokoll (device/protocol.go, SamNeo2Protocol) ist gegen den offiziellen
// Buttplug-Rust-Quellcode verifiziert, siehe Kommentare dort für die genauen
// Quellenpfade.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/funfunpayer/SamNPlayer/device"
	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/player"
)

func main() {
	scriptPath := flag.String("script", "", "Pfad zur .funscript-Datei (Pflicht)")
	mock := flag.Bool("mock", false, "Kein BLE - Befehle nur auf der Konsole ausgeben")
	verbose := flag.Bool("v", false, "Ausführliche Statusausgaben während der Wiedergabe")
	tickMs := flag.Int64("tick", 50, "Resampling-Intervall in ms (kleiner = feiner, mehr BLE-Writes)")
	maxSpeed := flag.Float64("max-speed", 0.6, "Positionsänderung (pos/ms), die als volle Intensität gilt")
	syncModeStr := flag.String("sync", "independent",
		"Wie Vibration/Sog zueinander stehen: independent, synchronized, alternating")

	extendedOEnabled := flag.Bool("extended-o", true,
		"Extended-O per [Enter] während der Wiedergabe aktivieren")
	extendedOMin := flag.Float64("extended-o-min", 0.1, "Intensität während des Extended-O-Haltens (0-1)")
	extendedOHold := flag.Duration("extended-o-hold", 10*time.Second, "Haltedauer für Extended-O (z.B. 10s, 1m)")
	extendedORestore := flag.Duration("extended-o-restore", 500*time.Millisecond,
		"Rampzeit zurück auf vorheriges Niveau nach Extended-O (0 = sofort)")

	flag.Parse()

	if *scriptPath == "" {
		fmt.Fprintln(os.Stderr, "Fehler: --script ist erforderlich")
		flag.Usage()
		os.Exit(1)
	}

	syncMode, err := funscript.ParseSyncMode(*syncModeStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}

	script, err := funscript.Load(*scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler beim Laden des Skripts: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Skript geladen: %d Actions, Dauer %.1fs\n",
		len(script.Actions), float64(script.Duration())/1000)

	opts := funscript.DefaultMapOptions()
	opts.TickMs = *tickMs
	opts.MaxSpeed = *maxSpeed
	opts.Sync = syncMode
	frames := script.ToIntensityCurve(opts)
	fmt.Printf("In %d Steuer-Frames umgerechnet (alle %dms, sync=%s)\n", len(frames), *tickMs, syncMode)

	var dev device.Device
	if *mock {
		dev = device.NewMock(*verbose)
	} else {
		dev = device.NewSamNeo2(device.SamNeo2Protocol{})
	}

	// Ctrl+C sauber abfangen und Gerät stoppen statt hart abzubrechen.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nAbbruch angefordert, stoppe...")
		cancel()
	}()

	connectCtx, connectCancel := context.WithTimeout(ctx, 20*time.Second)
	defer connectCancel()
	if err := dev.Connect(connectCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Verbindung fehlgeschlagen: %v\n", err)
		os.Exit(1)
	}
	defer dev.Disconnect()

	p := player.New(dev)
	if *verbose {
		p.LogEvery = time.Second
	}

	if *extendedOEnabled {
		fmt.Printf("Extended-O: [Enter] drücken für %.0f%% Intensität, %s halten\n",
			*extendedOMin*100, *extendedOHold)
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				p.TriggerExtendedO(player.ExtendedOOptions{
					MinLevel:        *extendedOMin,
					HoldDuration:    *extendedOHold,
					RestoreDuration: *extendedORestore,
				})
			}
		}()
	}

	fmt.Println("Wiedergabe startet...")
	if err := p.Play(ctx, frames); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "Fehler während der Wiedergabe: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Fertig.")
}
