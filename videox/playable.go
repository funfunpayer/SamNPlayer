package videox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PlayableCodecs are typically decoded by Chromium/WebView2 and WebKitGTK.
var PlayableCodecs = map[string]bool{
	"h264": true,
	"avc1": true,
	"vp8":  true,
	"vp9":  true,
	"av1":  true,
	"vp09": true,
	"av01": true,
}

// LikelyPlayable reports whether Info.Codec is usually playable in the
// embedded webview <video> element without a proxy.
func (i Info) LikelyPlayable() bool {
	c := strings.ToLower(strings.TrimSpace(i.Codec))
	if c == "" {
		return false
	}
	if PlayableCodecs[c] {
		return true
	}
	// Some probes report long names.
	for _, p := range []string{"h264", "avc", "vp8", "vp9", "av1"} {
		if strings.Contains(c, p) {
			return true
		}
	}
	return false
}

// ProxyPath returns a sibling cache path for an H.264/AAC MP4 proxy.
func ProxyPath(src string) string {
	dir := filepath.Dir(src)
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	return filepath.Join(dir, "."+base+".samnplayer-h264.mp4")
}

// EnsurePlayableProxy returns src if already likely playable, otherwise
// creates (or reuses) an H.264+AAC MP4 via ffmpeg next to the source.
// Requires ffmpeg on PATH. onProgress may be nil.
func EnsurePlayableProxy(ctx context.Context, src string, onProgress func(string)) (outPath string, converted bool, err error) {
	info, err := Probe(ctx, src)
	if err != nil {
		return "", false, fmt.Errorf("probe: %w", err)
	}
	if info.LikelyPlayable() {
		ext := strings.ToLower(filepath.Ext(src))
		// MKV/AVI with H.264 still often fail in webview — prefer remux/proxy
		// for non-mp4/webm containers even when codec looks fine.
		if ext == ".mp4" || ext == ".m4v" || ext == ".webm" || ext == ".mov" {
			return src, false, nil
		}
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", false, fmt.Errorf("ffmpeg fehlt — Codec %q vermutlich nicht abspielbar", info.Codec)
	}
	dst := ProxyPath(src)
	if st, err := os.Stat(dst); err == nil && !st.IsDir() && st.Size() > 1024 {
		// Reuse if newer than source.
		srcSt, _ := os.Stat(src)
		if srcSt == nil || !srcSt.ModTime().After(st.ModTime()) {
			if onProgress != nil {
				onProgress("Vorhandene Abspiel-Kopie: " + filepath.Base(dst))
			}
			return dst, true, nil
		}
	}
	if onProgress != nil {
		onProgress(fmt.Sprintf("Konvertiere nach H.264/AAC (%s → mp4)…", info.Codec))
	}
	// Faststart for progressive HTTP seek; scale down only if huge.
	args := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", src,
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "20",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac", "-b:a", "160k",
		"-movflags", "+faststart",
		dst,
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if b, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(dst)
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = err.Error()
		}
		return "", false, fmt.Errorf("ffmpeg: %s", msg)
	}
	if onProgress != nil {
		onProgress("Fertig: " + filepath.Base(dst))
	}
	return dst, true, nil
}

// ProbeTimeout is the default probe budget for UI calls.
const ProbeTimeout = 8 * time.Second
