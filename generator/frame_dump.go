package generator

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/funfunpayer/SamNPlayer/videox"
)

// DumpFrameAt extracts a single PNG frame at timeSec (seconds from start)
// via ffmpeg — no Python. Used for seek-past-black-intro in the GUI.
func DumpFrameAt(videoPath, outputPNG string, timeSec float64) (width, height int, err error) {
	if timeSec < 0 {
		timeSec = 0
	}
	cmd, err := videox.CommandContext(nil,
		"-v", "error", "-y",
		"-ss", strconv.FormatFloat(timeSec, 'f', 3, 64),
		"-i", videoPath,
		"-frames:v", "1",
		"-f", "image2",
		outputPNG,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("generator: ffmpeg nicht gefunden")
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("generator: Frame bei %.2fs fehlgeschlagen: %w\n%s", timeSec, err, string(out))
	}
	probe, err := videox.ProbeCommandContext(nil, "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=p=0:s=x", outputPNG)
	if err != nil {
		w, h, err2 := pngSize(outputPNG)
		if err2 != nil {
			return 0, 0, fmt.Errorf("generator: Framegröße unbekannt: %v / %v", err, err2)
		}
		return w, h, nil
	}
	info, err := probe.CombinedOutput()
	if err != nil {
		w, h, err2 := pngSize(outputPNG)
		if err2 != nil {
			return 0, 0, fmt.Errorf("generator: Framegröße unbekannt: %v / %v", err, err2)
		}
		return w, h, nil
	}
	parts := strings.Split(strings.TrimSpace(string(info)), "x")
	if len(parts) != 2 {
		w, h, err2 := pngSize(outputPNG)
		if err2 != nil {
			return 0, 0, fmt.Errorf("generator: Framegröße nicht lesbar: %s", string(info))
		}
		return w, h, nil
	}
	w, _ := strconv.Atoi(parts[0])
	h, _ := strconv.Atoi(parts[1])
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("generator: ungültige Framegröße %dx%d", w, h)
	}
	return w, h, nil
}

// DumpFirstFrameFast is ffmpeg-based first-frame dump (no Python). Prefer
// this over DumpFirstFrame when Python deps are unavailable.
func DumpFirstFrameFast(videoPath, outputPNG string) (width, height int, err error) {
	return DumpFrameAt(videoPath, outputPNG, 0)
}

func pngSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	var header [24]byte
	if _, err := f.Read(header[:]); err != nil {
		return 0, 0, err
	}
	// PNG IHDR: bytes 16-23 are width/height big-endian.
	if string(header[1:4]) != "PNG" {
		return 0, 0, fmt.Errorf("kein PNG")
	}
	w := int(header[16])<<24 | int(header[17])<<16 | int(header[18])<<8 | int(header[19])
	h := int(header[20])<<24 | int(header[21])<<16 | int(header[22])<<8 | int(header[23])
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("PNG-Größe %dx%d", w, h)
	}
	return w, h, nil
}
