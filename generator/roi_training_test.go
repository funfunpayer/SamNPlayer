package generator

import (
	"strings"
	"testing"
)

func TestBuildBootstrapArgsSingleRegion(t *testing.T) {
	regions := []RoiTrainingRegion{{ROI: ROI{X: 1, Y: 2, W: 3, H: 4}, ClassName: "brust"}}
	args := buildBootstrapArgs("script.py", "v.mp4", regions, "/data", "")
	joined := strings.Join(args, " ")
	for _, want := range []string{"--video v.mp4", "--roi 1,2,3,4", "--class-name brust", "--output-dir /data"} {
		if !strings.Contains(joined, want) {
			t.Errorf("Argument %q fehlt in: %s", want, joined)
		}
	}
	for _, unwanted := range []string{"--roi2", "--class-name2", "--sample-prefix"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("unerwartetes Argument %q ohne 2. Region/Präfix: %s", unwanted, joined)
		}
	}
}

func TestBuildBootstrapArgsStartSeconds(t *testing.T) {
	regions := []RoiTrainingRegion{{ROI: ROI{X: 1, Y: 2, W: 3, H: 4}, ClassName: "brust"}}
	args := buildBootstrapArgsOpts("script.py", "v.mp4", regions, "/data", "", 12, 4.5)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--start-seconds 4.500") {
		t.Errorf("start-seconds fehlt in: %s", joined)
	}
	args0 := buildBootstrapArgsOpts("script.py", "v.mp4", regions, "/data", "", 12, 0)
	if strings.Contains(strings.Join(args0, " "), "--start-seconds") {
		t.Errorf("start-seconds bei 0 unerwartet: %v", args0)
	}
}

func TestBuildTrainArgsDefaults(t *testing.T) {
	// epochs<=0 und device="" sollen kein Flag erzeugen - train_yolo_model.py
	// hat dafür eigene Standardwerte (100 Epochen, device="auto"), die dann
	// nicht durch ein leeres/nullwertiges Flag überschrieben werden sollen.
	args := buildTrainArgs("script.py", "/data", "/models/roi.onnx", 0, "")
	joined := strings.Join(args, " ")
	for _, want := range []string{"--dataset-dir /data", "--output /models/roi.onnx"} {
		if !strings.Contains(joined, want) {
			t.Errorf("Argument %q fehlt in: %s", want, joined)
		}
	}
	for _, unwanted := range []string{"--epochs", "--device"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("unerwartetes Argument %q ohne explizite Angabe: %s", unwanted, joined)
		}
	}
}

func TestBuildTrainArgsExplicitValues(t *testing.T) {
	args := buildTrainArgs("script.py", "/data", "/models/roi.onnx", 50, "cpu")
	joined := strings.Join(args, " ")
	for _, want := range []string{"--epochs 50", "--device cpu"} {
		if !strings.Contains(joined, want) {
			t.Errorf("Argument %q fehlt in: %s", want, joined)
		}
	}
}

func TestRoiArgFormatsAsCommaSeparated(t *testing.T) {
	got := roiArg(ROI{X: 10, Y: 20, W: 30, H: 40})
	if got != "10,20,30,40" {
		t.Fatalf("erwartete '10,20,30,40', bekam %q", got)
	}
}

func TestDefaultRoiDatasetDirIsUnderSamNPlayerNamespace(t *testing.T) {
	got := DefaultRoiDatasetDir()
	if got == "" {
		t.Skip("os.UserConfigDir() liefert in dieser Umgebung keinen Pfad")
	}
	if !strings.Contains(got, "SamNPlayer") || !strings.HasSuffix(got, "roi_training_dataset") {
		t.Fatalf("erwarte einen Pfad unter .../SamNPlayer/roi_training_dataset, bekam %q", got)
	}
}
