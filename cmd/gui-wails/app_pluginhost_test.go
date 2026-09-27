package main

import (
	"testing"

	"github.com/funfunpayer/SamNPlayer/license"
	"github.com/funfunpayer/SamNPlayer/pluginhost"
)

func TestLicenseAllowsVirtualPersonWhileOff(t *testing.T) {
	old := license.Enforcement
	license.Enforcement = false
	defer func() { license.Enforcement = old }()

	a := &App{vpHost: pluginhost.NewSlot()}
	if !a.LicenseAllowsVirtualPerson() {
		t.Fatal("enforcement off must allow virtual_person")
	}
	st, err := a.EnableVirtualPersonHost()
	if err != nil {
		t.Fatal(err)
	}
	if !st.Running || st.FeatureID != pluginhost.PluginIDVirtualPerson {
		t.Fatalf("%+v", st)
	}
	a.tickVirtualPersonHost(42, 0.4, 0.2)
	if a.vpHost.TickCount() != 1 || a.vpHost.NowMs() != 42 {
		t.Fatalf("tick count=%d now=%d", a.vpHost.TickCount(), a.vpHost.NowMs())
	}
	off := a.DisableVirtualPersonHost()
	if off.Running {
		t.Fatal("expected stopped")
	}
}

func TestEnableVirtualPersonHostDeniedWhenSharp(t *testing.T) {
	old := license.Enforcement
	license.Enforcement = true
	defer func() { license.Enforcement = old }()
	t.Setenv("SAMN_LICENSE_OFF", "")

	a := &App{vpHost: pluginhost.NewSlot()}
	if a.LicenseAllowsVirtualPerson() {
		t.Fatal("no key + enforcement on must deny")
	}
	_, err := a.EnableVirtualPersonHost()
	if err == nil {
		t.Fatal("expected license error")
	}
	st := a.VirtualPersonHostStatus()
	if st.Allowed || st.Running {
		t.Fatalf("%+v", st)
	}
}
