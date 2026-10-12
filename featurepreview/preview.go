// Package featurepreview provides a seven-day opt-in Community feature trial.
// It never applies to scripts, videos, playback, or other user-owned data.
package featurepreview

import (
 "errors"
 "strings"
 "time"
)

const Duration = 7 * 24 * time.Hour
const ClockTolerance = 5 * time.Minute

var (
 ErrNotStarted = errors.New("preview has not been activated")
 ErrExpired = errors.New("seven-day preview expired")
 ErrClockRollback = errors.New("device clock moved backwards: preview cannot be verified")
 ErrInvalidState = errors.New("invalid preview state")
)

// State belongs to one optional feature. Callers persist this metadata outside
// projects/scripts. StartedAt must never be reset merely by reinstall/upgrade.
type State struct {
 FeatureID string `json:"featureId"`
 StartedAt time.Time `json:"startedAt"`
 LastSeenAt time.Time `json:"lastSeenAt"`
}

// Activate should be called only in response to an explicit user action.
// Whether a feature is eligible is decided by the edition/feature registry.
func Activate(featureID string, now time.Time) (State, error) {
 if strings.TrimSpace(featureID) == "" || now.IsZero() { return State{}, ErrInvalidState }
 now = now.UTC()
 return State{FeatureID: featureID, StartedAt: now, LastSeenAt: now}, nil
}

// Check evaluates a persisted preview; exactly at the deadline it is expired.
// It returns an updated LastSeenAt that the caller must persist atomically.
// Errors do not modify the state. The check alone never edits user content.
func Check(s State, now time.Time) (State, time.Duration, error) {
 if strings.TrimSpace(s.FeatureID) == "" || s.StartedAt.IsZero() || s.LastSeenAt.IsZero() || now.IsZero() ||
   s.LastSeenAt.Before(s.StartedAt) { return s, 0, ErrInvalidState }
 now = now.UTC()
 if now.Before(s.StartedAt.Add(-ClockTolerance)) || now.Before(s.LastSeenAt.Add(-ClockTolerance)) {
   return s, 0, ErrClockRollback
 }
 end := s.StartedAt.Add(Duration)
 if !now.Before(end) { return s, 0, ErrExpired }
 if now.After(s.LastSeenAt) { s.LastSeenAt = now }
 return s, end.Sub(now), nil
}

// Gate applies feature-specific eligibility and preserves an existing trial.
// Calling it repeatedly never grants another seven days.
func Gate(featureID string, state *State, now time.Time) (State, time.Duration, error) {
 if state == nil { return State{}, 0, ErrNotStarted }
 if state.FeatureID != featureID { return *state, 0, ErrInvalidState }
 return Check(*state, now)
}
