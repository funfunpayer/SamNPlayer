package update

import (
	"runtime"
	"testing"
)

func TestDescribe(t *testing.T) {
	got := Describe()
	if got == "" {
		t.Fatal("empty describe")
	}
	if Version == "dev" && got != BaseVersion+"-dev" {
		t.Fatalf("dev describe=%q want %s-dev", got, BaseVersion)
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		cur, lat string
		want     bool
	}{
		{"dev", "v0.1.0", true},
		{"v0.4.0", "v0.5.0", true},
		{"v0.5.0", "v0.5.0", false},
		{"v0.5.1", "v0.5.0", false},
		{"0.5.0", "v0.6.0", true},
	}
	for _, c := range cases {
		if got := IsNewer(c.cur, c.lat); got != c.want {
			t.Errorf("IsNewer(%q,%q)=%v want %v", c.cur, c.lat, got, c.want)
		}
	}
}

func TestAssetForThisPlatform(t *testing.T) {
	suffix := "-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		suffix += ".exe"
	}
	rel := &Release{Assets: []Asset{
		{Name: "SamNPlayer-gui" + suffix},
		{Name: "SamNPlayer-cli" + suffix},
		{Name: "other-file.txt"},
	}}
	gui, ok := rel.AssetForThisPlatform("gui")
	if !ok || gui.Name != "SamNPlayer-gui"+suffix {
		t.Fatalf("gui asset: %+v ok=%v", gui, ok)
	}
	cli, ok := rel.AssetForThisPlatform("cli")
	if !ok || cli.Name != "SamNPlayer-cli"+suffix {
		t.Fatalf("cli asset: %+v ok=%v", cli, ok)
	}
	if _, ok := rel.AssetForThisPlatform("missing"); ok {
		t.Fatal("expected no asset for missing kind")
	}
}

func TestValidateGitHubAssetURL(t *testing.T) {
	good := "https://github.com/funfunpayer/SamNPlayer/releases/download/v0.5.0/SamNPlayer-gui-linux-amd64"
	if err := validateGitHubAssetURL(good); err != nil {
		t.Fatal(err)
	}
	if err := validateGitHubAssetURL("https://evil.example/x"); err == nil {
		t.Fatal("expected reject non-github host")
	}
}
