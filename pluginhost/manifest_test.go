package pluginhost

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifestAndDiscover(t *testing.T) {
	root := t.TempDir()
	packDir := filepath.Join(root, "sample_pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{
  "apiVersion": 1,
  "id": "sample_pack",
  "name": "Sample Pack",
  "version": "0.1.0",
  "requiresFeature": "sample_feature",
  "entry": {
    "kind": "asset_pack",
    "webRoot": "web",
    "assetsRoot": "assets"
  }
}`
	if err := os.WriteFile(filepath.Join(packDir, ManifestFile), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(packDir)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "sample_pack" || m.APIVersion != 1 {
		t.Fatalf("manifest %+v", m)
	}
	packs, err := DiscoverPacks(root)
	if err != nil || len(packs) != 1 {
		t.Fatalf("discover: %v packs=%d", err, len(packs))
	}
	got, ok, err := FindPackByID(root, "sample_pack")
	if err != nil || !ok || got.Manifest.Version != "0.1.0" {
		t.Fatalf("find: ok=%v err=%v got=%+v", ok, err, got)
	}
}

func TestRejectBadAPIVersion(t *testing.T) {
	dir := t.TempDir()
	raw := `{"apiVersion":99,"id":"x","name":"X","version":"1","entry":{"kind":"asset_pack"}}`
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(dir); err == nil {
		t.Fatal("expected apiVersion error")
	}
}

func TestInstallPackDir(t *testing.T) {
	src := t.TempDir()
	raw := `{
  "apiVersion": 1,
  "id": "sample_pack",
  "name": "Sample Pack",
  "version": "0.2.0",
  "entry": {"kind": "asset_pack", "webRoot": "web"}
}`
	if err := os.WriteFile(filepath.Join(src, ManifestFile), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "note.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	destRoot := t.TempDir()
	pack, err := InstallPackDir(destRoot, src)
	if err != nil {
		t.Fatal(err)
	}
	if pack.Manifest.Version != "0.2.0" {
		t.Fatalf("%+v", pack)
	}
	if _, err := os.Stat(filepath.Join(pack.Root, "note.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestStatusReflectsPack(t *testing.T) {
	s := NewSlot()
	s.SetPluginsDir("/tmp/plugins")
	st := s.Status(true)
	if st.PackFound || st.Stage != Stage {
		t.Fatalf("%+v", st)
	}
	p := &Pack{
		Root: "/tmp/plugins/sample_pack",
		Manifest: Manifest{
			APIVersion: 1,
			ID:         "sample_pack",
			Name:       "Sample Pack",
			Version:    "0.1.0",
			Entry:      ManifestEntry{Kind: "asset_pack"},
		},
	}
	s.BindPack(p)
	st = s.Status(true)
	if !st.PackFound || st.PackVersion != "0.1.0" {
		t.Fatalf("%+v", st)
	}
	if err := s.Enable(true); err != nil {
		t.Fatal(err)
	}
	st = s.Status(true)
	if !st.Running || st.Message == "" {
		t.Fatalf("%+v", st)
	}
}
