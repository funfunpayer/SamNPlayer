package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func writeRoiSample(t *testing.T, datasetDir, split, name, labelBody string) string {
	t.Helper()
	imgDir := filepath.Join(datasetDir, "images", split)
	lblDir := filepath.Join(datasetDir, "labels", split)
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(lblDir, 0o755); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(imgDir, name+".jpg")
	if err := os.WriteFile(imagePath, []byte("fake jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	if labelBody != "" {
		if err := os.WriteFile(filepath.Join(lblDir, name+".txt"), []byte(labelBody), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return imagePath
}

func TestRoiLabelPathForImage(t *testing.T) {
	imagePath := filepath.Join("dataset", "images", "train", "clip_000001.jpg")
	got, err := roiLabelPathForImage(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("dataset", "labels", "train", "clip_000001.txt")
	if got != want {
		t.Fatalf("erwartete %q, bekam %q", want, got)
	}
}

func TestRoiLabelPathForImageWithoutImagesDirIsAnError(t *testing.T) {
	if _, err := roiLabelPathForImage(filepath.Join("dataset", "clip.jpg")); err == nil {
		t.Fatal("erwartete einen Fehler ohne 'images'-Ordner im Pfad")
	}
}

func TestReadRoiLabelFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	names := map[int]string{0: "eichel", 1: "brustwarze"}

	boxes := []RoiTrainingBox{
		{ClassID: 0, ClassName: "eichel", XC: 0.5, YC: 0.5, W: 0.2, H: 0.2},
		{ClassID: 1, ClassName: "brustwarze", XC: 0.3, YC: 0.4, W: 0.1, H: 0.1},
	}
	if err := writeRoiLabelFile(path, boxes); err != nil {
		t.Fatal(err)
	}
	got, err := readRoiLabelFile(path, names)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("erwartete 2 Boxen, bekam %d: %+v", len(got), got)
	}
	if got[0].ClassName != "eichel" || got[1].ClassName != "brustwarze" {
		t.Fatalf("Klassennamen wurden nicht richtig aus classes.json aufgelöst: %+v", got)
	}
	if got[0].XC != 0.5 || got[0].YC != 0.5 {
		t.Fatalf("Koordinaten stimmen nach dem Rundlauf nicht: %+v", got[0])
	}
}

func TestReadRoiLabelFileMissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	got, err := readRoiLabelFile(filepath.Join(dir, "fehlt.txt"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("erwartete leere Liste ohne Labeldatei, bekam %+v", got)
	}
}

func TestReadRoiLabelFileSkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(path, []byte("0 0.5 0.5 0.2 0.2\nkaputte zeile\n1 0.1 0.1 0.05 0.05\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readRoiLabelFile(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("kaputte Zeile hätte übersprungen werden müssen, bekam %d Boxen: %+v", len(got), got)
	}
}

func TestListRoiTrainingSamplesFiltersByPrefix(t *testing.T) {
	dir := t.TempDir()
	writeRoiSample(t, dir, "train", "videoA_000000", "0 0.5 0.5 0.2 0.2\n")
	writeRoiSample(t, dir, "train", "videoA_000001", "0 0.4 0.4 0.2 0.2\n")
	writeRoiSample(t, dir, "val", "videoB_000000", "1 0.6 0.6 0.1 0.1\n")

	a := NewApp()

	all, err := a.ListRoiTrainingSamples(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("ohne Präfix erwartete 3 Beispiele (train+val), bekam %d: %+v", len(all), all)
	}

	filtered, err := a.ListRoiTrainingSamples(dir, "videoA")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Fatalf("mit Präfix 'videoA' erwartete 2 Beispiele, bekam %d: %+v", len(filtered), filtered)
	}
	for _, s := range filtered {
		if s.Split != "train" {
			t.Fatalf("videoA-Beispiele sollten aus 'train' kommen, bekam %+v", s)
		}
	}
}

func TestUpdateAndDiscardRoiTrainingSample(t *testing.T) {
	dir := t.TempDir()
	writeRoiSample(t, dir, "train", "clip_000000", "0 0.5 0.5 0.2 0.2\n")

	a := NewApp()

	newBoxes := []RoiTrainingBox{{ClassID: 0, XC: 0.6, YC: 0.6, W: 0.15, H: 0.15}}
	if err := a.UpdateRoiTrainingSample(dir, "train", "clip_000000.jpg", newBoxes); err != nil {
		t.Fatal(err)
	}
	samples, err := a.ListRoiTrainingSamples(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 1 || len(samples[0].Boxes) != 1 || samples[0].Boxes[0].XC != 0.6 {
		t.Fatalf("Korrektur wurde nicht übernommen: %+v", samples)
	}

	if err := a.DiscardRoiTrainingSample(dir, "train", "clip_000000.jpg"); err != nil {
		t.Fatal(err)
	}
	afterDiscard, err := a.ListRoiTrainingSamples(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(afterDiscard) != 0 {
		t.Fatalf("erwartete leere Liste nach dem Verwerfen, bekam %+v", afterDiscard)
	}
	if _, err := os.Stat(filepath.Join(dir, "images", "train", "clip_000000.jpg")); !os.IsNotExist(err) {
		t.Fatal("discard sollte auch die Bilddatei entfernen")
	}
}

func TestGetRoiDatasetSummaryCountsPerClassAndSplit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "classes.json"), []byte(`{"eichel":0,"brustwarze":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	writeRoiSample(t, dir, "train", "a", "0 0.5 0.5 0.2 0.2\n1 0.3 0.3 0.1 0.1\n")
	writeRoiSample(t, dir, "train", "b", "0 0.5 0.5 0.2 0.2\n")
	writeRoiSample(t, dir, "val", "c", "1 0.3 0.3 0.1 0.1\n")

	a := NewApp()
	summary, err := a.GetRoiDatasetSummary(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Classes) != 2 {
		t.Fatalf("erwartete 2 Klassen, bekam %+v", summary.Classes)
	}
	byName := map[string]RoiClassCount{}
	for _, c := range summary.Classes {
		byName[c.ClassName] = c
	}
	if byName["eichel"].TrainCount != 2 || byName["eichel"].ValCount != 0 {
		t.Fatalf("falsche Zählung für 'eichel': %+v", byName["eichel"])
	}
	if byName["brustwarze"].TrainCount != 1 || byName["brustwarze"].ValCount != 1 {
		t.Fatalf("falsche Zählung für 'brustwarze': %+v", byName["brustwarze"])
	}
}

func TestGetRoiDatasetSummaryWithoutClassesJsonIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	a := NewApp()
	summary, err := a.GetRoiDatasetSummary(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Classes) != 0 {
		t.Fatalf("erwartete leere Klassenliste ohne classes.json, bekam %+v", summary.Classes)
	}
}

func TestGetRoiTrainingSampleImageReturnsBase64(t *testing.T) {
	dir := t.TempDir()
	imagePath := writeRoiSample(t, dir, "train", "clip_000000", "")

	a := NewApp()
	b64, err := a.GetRoiTrainingSampleImage(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("Ergebnis ist kein gültiges Base64: %v", err)
	}
	if string(decoded) != "fake jpg" {
		t.Fatalf("Base64 dekodiert nicht auf den Dateiinhalt: %q", decoded)
	}
}

func TestGetRoiTrainingSampleImageMissingFileIsAnError(t *testing.T) {
	dir := t.TempDir()
	a := NewApp()
	if _, err := a.GetRoiTrainingSampleImage(filepath.Join(dir, "fehlt.jpg")); err == nil {
		t.Fatal("erwartete einen Fehler für eine fehlende Bilddatei")
	}
}

func TestRoiTrainingSamplePrefixIsFilenameSafeAndUnique(t *testing.T) {
	p1 := roiTrainingSamplePrefix(`C:\videos\Mein Clip! (2026).mp4`)
	p2 := roiTrainingSamplePrefix(`C:\videos\Mein Clip! (2026).mp4`)
	for _, r := range p1 {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
			t.Fatalf("Präfix enthält ein dateinamen-unsicheres Zeichen: %q in %q", r, p1)
		}
	}
	if p1 == p2 {
		t.Fatal("zwei Aufrufe für denselben Videopfad sollten unterschiedliche Präfixe liefern (Zeitkomponente)")
	}
}
