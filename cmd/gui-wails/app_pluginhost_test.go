package main

import (
	"testing"

	"github.com/funfunpayer/SamNPlayer/license"
	"github.com/funfunpayer/SamNPlayer/pluginhost"
	"github.com/funfunpayer/SamNPlayer/virtualperson"
)

func TestLicenseAllowsVirtualPersonWhileOff(t *testing.T) {
	old := license.Enforcement
	license.Enforcement = false
	defer func() { license.Enforcement = old }()

	a := &App{}
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
	if st.Stage != pluginhost.Stage {
		t.Fatalf("stage=%s want %s", st.Stage, pluginhost.Stage)
	}
	a.tickVirtualPersonHost(42, 0.4, 0.2)
	if a.vpHost.TickCount() != 1 || a.vpHost.NowMs() != 42 {
		t.Fatalf("tick count=%d now=%d", a.vpHost.TickCount(), a.vpHost.NowMs())
	}
	if a.vpClockMs.Load() != 42 {
		t.Fatalf("vpClockMs=%d", a.vpClockMs.Load())
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

	a := &App{}
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

func TestVirtualPersonSceneStepsRequireHost(t *testing.T) {
	old := license.Enforcement
	license.Enforcement = false
	defer func() { license.Enforcement = old }()

	a := &App{}
	if err := a.VirtualPersonGiveDildo(); err == nil {
		t.Fatal("give without enable should fail")
	}
	if _, err := a.EnableVirtualPersonHost(); err != nil {
		t.Fatal(err)
	}
	if err := a.VirtualPersonGiveDildo(); err != nil {
		t.Fatal(err)
	}
	if !a.vpPlugin.Inventory().Has(virtualperson.PropDildo) {
		t.Fatal("expected dildo")
	}
	if err := a.VirtualPersonStartTitjob(0.5, 0); err != nil {
		t.Fatal(err)
	}
	if a.vpPlugin.Catalog().State() != virtualperson.StateTitjobActive {
		t.Fatalf("state=%s", a.vpPlugin.Catalog().State())
	}
	a.tickVirtualPersonHost(1000, 0.1, 0.1)
	if a.vpHost.TickCount() != 1 {
		t.Fatalf("ticks=%d", a.vpHost.TickCount())
	}
}

func TestTickNoOpWhenHostDisabled(t *testing.T) {
	a := &App{}
	a.tickVirtualPersonHost(99, 0.5, 0.5)
	if a.vpHost != nil && a.vpHost.TickCount() != 0 {
		t.Fatal("disabled tick must not count")
	}
}
