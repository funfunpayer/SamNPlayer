//go:build cgo && opencv

package trackcv

import "github.com/funfunpayer/SamNPlayer/generator/trackutil"

func (o Options) appearanceMemoryEnabled() bool {
	return o.AppearanceMemory
}

// recoverOrCoast: on a successful update, remember + reset coast. On miss,
// try appearance reacquire (same partner/tip ID slot — re-init tracker), else
// leave ok=false for the caller to coast or exclude.
func recoverOrCoast(
	cap *VideoCapture,
	gray *Gray,
	tr *Tracker,
	mem *appearanceMemory,
	coast *trackutil.Coast,
	newBox Rect,
	ok bool,
	prev Rect,
	frameIdx int,
) (okOut bool, box Rect, tracker *Tracker) {
	tracker = tr
	if ok {
		box = newBox
		coast.ObserveOK(box.X, box.Y, box.W, box.H)
		if mem != nil && frameIdx%rememberEveryNFrames == 0 && gray != nil {
			mem.remember(gray, box)
		}
		return true, box, tracker
	}
	if mem != nil && gray != nil {
		if found, reacquired := mem.reacquire(gray); reacquired {
			if tracker != nil {
				tracker.Close()
			}
			tracker = NewTracker()
			tracker.Init(cap, found)
			coast.Reset()
			coast.ObserveOK(found.X, found.Y, found.W, found.H)
			return true, found, tracker
		}
	}
	return false, prev, tracker
}
