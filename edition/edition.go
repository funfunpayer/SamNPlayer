// Package edition is the build switch between the customer build and the
// internal studio build (Owner 11 Oct 2026).
//
// The customer build (default, what releases ship) runs SamNPlayer with all
// player, generator and AI-assist features. AI training and learning tools —
// AI Train, the golden-clip benchmark, learning-data export, model training,
// candidate review for datasets — are only in the studio build:
//
//	go build -tags studio ./cmd/cli
//	wails build -tags "studio ..."
package edition

// Studio reports whether this binary is the internal studio build.
func Studio() bool { return studio }

// Name is "studio" or "customer" (logs, About).
func Name() string {
	if studio {
		return "studio"
	}
	return "customer"
}
