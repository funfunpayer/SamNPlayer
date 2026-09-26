package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/logging"
	"github.com/funfunpayer/SamNPlayer/player"
	"github.com/funfunpayer/SamNPlayer/sam"
	"github.com/funfunpayer/SamNPlayer/samn"
)

type PlaybackOptions struct {
	Mock                    bool    `json:"mock"`
	SyncMode                string  `json:"syncMode"`
	TickMs                  int64   `json:"tickMs"`
	MaxSpeed                float64 `json:"maxSpeed"`
	Smoothing               float64 `json:"smoothing"`
	SoftStartMs             int     `json:"softStartMs"`
	UseVideoSync            bool    `json:"useVideoSync"`
	ExtendedOEnabled        bool    `json:"extendedOEnabled"`
	ExtendedOMin            float64 `json:"extendedOMin"`
	ExtendedOHoldS          float64 `json:"extendedOHoldS"`
	ExtendedORestoreMs      float64 `json:"extendedORestoreMs"`
	DisableContactVibration bool    `json:"disableContactVibration"`
	ContactIntensityScale float64 `json:"contactIntensityScale"`
	ContactExtraSmooth float64 `json:"contactExtraSmooth"`
	ContactVibrationSpan  float64 `json:"contactVibrationSpan"`
	ContactVibrationCurve string  `json:"contactVibrationCurve"`
}

// RESTORE_INCOMPLETE - full file will be restored in next commit
