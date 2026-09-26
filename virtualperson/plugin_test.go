package virtualperson_test

import (
	"context"
	"testing"

	"github.com/funfunpayer/SamNPlayer/device"
	"github.com/funfunpayer/SamNPlayer/virtualperson"
)

type testHost struct {
	now   int64
	poses []virtualperson.PoseSample
	dev   virtualperson.DeviceBridge
}

func (h *testHost) Device() virtualperson.DeviceBridge { return h.dev }
func (h *testHost) NowMs() int64                       { return h.now }
func (h *testHost) EmitAnimation(pose virtualperson.PoseSample) {
	h.poses = append(h.poses, pose)
}

func TestTitjobDildoSceneGraph(t *testing.T) {
	host := &testHost{now: 1000, dev: virtualperson.NullDevice{}}
	p := virtualperson.NewPlugin(host)

	if err := p.StartTitjob(0.5, 10); err == nil {
		t.Fatal("titjob without dildo should fail")
	}
	if err := p.GiveDildo(); err != nil {
		t.Fatal(err)
	}
	if !p.Inventory().Has(virtualperson.PropDildo) {
		t.Fatal("expected dildo in inventory")
	}
	if err := p.StartTitjob(0.6, 0); err != nil {
		t.Fatal(err)
	}
	if p.Catalog().State() != virtualperson.StateTitjobActive {
		t.Fatalf("state=%s", p.Catalog().State())
	}

	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := p.Tick(10, 0.1, 0.1); err != nil {
		t.Fatal(err)
	}
	if len(host.poses) != 1 {
		t.Fatalf("poses=%d", len(host.poses))
	}
	pose := host.poses[0]
	if len(pose.Props) == 0 || pose.Props[0].PropID != virtualperson.PropDildo {
		t.Fatalf("props=%v", pose.Props)
	}
	if pose.Props[0].Socket != virtualperson.SocketChest {
		t.Fatalf("socket=%s want chest", pose.Props[0].Socket)
	}
	// Activity should win over funscript (stroke near titjob range, not 10).
	if pose.Channels.Stroke < 15 {
		t.Fatalf("expected activity stroke, got %v", pose.Channels.Stroke)
	}
	p.Stop()
}

func TestSampleAtInterpolation(t *testing.T) {
	actions := []virtualperson.FunscriptAction{
		{At: 0, Pos: 0},
		{At: 1000, Pos: 100},
	}
	got := virtualperson.SampleAt(actions, 500)
	if got < 49 || got > 51 {
		t.Fatalf("mid=%v", got)
	}
}

func TestDeviceBridgeFromMock(t *testing.T) {
	var mock device.Device = device.NewMock(false)
	b := virtualperson.DeviceBridgeFrom(mock)
	if b == nil {
		t.Fatal("nil bridge")
	}
	ctx := context.Background()
	if err := b.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err := b.SetVibration(0.5); err != nil {
		t.Fatal(err)
	}
	if err := b.Stop(); err != nil {
		t.Fatal(err)
	}
	_ = b.Disconnect()
}

func TestParsePropAndActivityTags(t *testing.T) {
	text := "Sure.\n[prop: give=dildo]\n[activity: id=titjob_dildo intensity=0.5 duration=30s]"
	if virtualperson.ParsePropGiveTag(text) != virtualperson.PropDildo {
		t.Fatal("prop tag")
	}
	act := virtualperson.ParseActivityTag(text)
	if act == nil || act.ID != virtualperson.ActivityTitjobDildo {
		t.Fatalf("activity=%v", act)
	}
}
