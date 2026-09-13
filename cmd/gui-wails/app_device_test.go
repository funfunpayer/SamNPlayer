package main

import "testing"

func TestDeviceTabFlow(t *testing.T) {
	a := NewApp()

	if st := a.GetDeviceStatus(); st.Connected {
		t.Fatal("frisch gestartet darf nichts verbunden sein")
	}
	if err := a.TestVibration(0.5); err == nil {
		t.Fatal("Test ohne Verbindung muss abgelehnt werden")
	}

	st, err := a.ConnectDevice(true)
	if err != nil || !st.Connected {
		t.Fatalf("Mock-Verbindung fehlgeschlagen: %v %+v", err, st)
	}
	if !st.Mock {
		t.Error("Mock-Flag nicht gesetzt")
	}

	if _, err := a.ConnectDevice(true); err == nil {
		t.Error("doppeltes Verbinden muss abgelehnt werden")
	}
	if err := a.TestVibration(0.5); err != nil {
		t.Errorf("Vibrationstest: %v", err)
	}
	if err := a.TestSuction(1.0); err != nil {
		t.Errorf("Sogtest: %v", err)
	}
	if err := a.TestStop(); err != nil {
		t.Errorf("Stop: %v", err)
	}

	// Kern: Testverbindung und Session dürfen sich nicht überlagern.
	if _, err := a.tryStartSession(); err == nil {
		t.Error("Session-Start bei offener Testverbindung muss abgelehnt werden")
	}

	if _, err := a.DisconnectDevice(); err != nil {
		t.Errorf("Trennen: %v", err)
	}
	if st := a.GetDeviceStatus(); st.Connected {
		t.Error("nach dem Trennen darf nichts verbunden sein")
	}

	// Nach dem Trennen muss eine Session wieder starten können.
	if _, err := a.tryStartSession(); err != nil {
		t.Errorf("Session nach Trennen: %v", err)
	}
	// Und dann sperrt die Session umgekehrt den Gerätetest.
	if _, err := a.ConnectDevice(true); err == nil {
		t.Error("Verbinden bei laufender Session muss abgelehnt werden")
	}
	a.endSession()
}
