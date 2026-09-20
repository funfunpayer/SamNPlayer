//go:build cgo && opencv && !windows

package trackcv

// progressReporter throttles OnProgress to ~100 updates (same cadence as
// generate_funscript.track_roi PROGRESS lines) so the GUI bar moves during
// long Go-native tracking instead of jumping 0→100 at the end.
type progressReporter struct {
	fn    func(done, total int)
	total int
	next  int
}

func newProgressReporter(fn func(done, total int), totalFrames int) *progressReporter {
	if fn == nil {
		return nil
	}
	if totalFrames < 0 {
		totalFrames = 0
	}
	return &progressReporter{fn: fn, total: totalFrames, next: 0}
}

func (p *progressReporter) report(done int) {
	if p == nil {
		return
	}
	if done < p.next && (p.total <= 0 || done < p.total) {
		return
	}
	p.fn(done, p.total)
	step := 10
	if p.total > 0 {
		step = p.total / 100
		if step < 1 {
			step = 1
		}
	}
	p.next = done + step
}
