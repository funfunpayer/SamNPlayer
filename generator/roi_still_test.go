package generator

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestStillTrainingGetsValSplit(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "src.jpg")
	if err := writeTestJPEG(imgPath, 64, 64); err != nil {
		t.Fatal(err)
	}
	regions := []RoiTrainingRegion{{
		ROI:       ROI{X: 10, Y: 10, W: 20, H: 20},
		ClassName: "hand",
	}}
	out := filepath.Join(dir, "ds")
	if _, err := AddStillTrainingSample(imgPath, regions, out, "a"); err != nil {
		t.Fatal(err)
	}
	if countImages(filepath.Join(out, "images", "train")) != 1 {
		t.Fatalf("first still should be train")
	}
	if _, err := AddStillTrainingSample(imgPath, regions, out, "b"); err != nil {
		t.Fatal(err)
	}
	if countImages(filepath.Join(out, "images", "val")) < 1 {
		t.Fatalf("second still should land in val when train already has one")
	}
}

func writeTestJPEG(path string, w, h int) error {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 40, G: 80, B: 120, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 80})
}
