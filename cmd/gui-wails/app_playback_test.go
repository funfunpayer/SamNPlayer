package main

import (
	"sync"
	"testing"
)

// Reproduziert eine echte Race: ReportVideoPosition (aus dem Frontend bei
// jedem "timeupdate" des Videos, ~4x/s) und StopPlayback (beim Stop-Klick)
// griffen beide unsynchronisiert auf a.videoPositionCh zu. Landete
// StopPlaybacks close() zwischen ReportVideoPositions nil-Prüfung und
// seinem send, panicte das mit "send on closed channel" - reproduziert vor
// dem Fix zuverlässig sowohl unter go test -race als auch (seltener, aber
// real) ganz ohne -race.
func TestReportVideoPositionStopPlaybackRace(t *testing.T) {
	a := NewApp()
	a.videoPositionCh = make(chan int64, 4)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			a.ReportVideoPosition(int64(i))
		}
	}()
	go func() {
		defer wg.Done()
		a.StopPlayback()
	}()
	wg.Wait()
}
