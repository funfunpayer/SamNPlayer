// Package virtualperson is the foundation for Virtual Persons inside SamNPlayer.
//
// Primary product surface: virtual toy props + scene activities.
// MVP: give_toy(dildo) → activity(titjob_dildo) → animation (+ prop sprites).
// Real Sam Neo 2 / Intiface output is an optional parallel channel (ToyHub).
//
// There is no host plugin marketplace yet. This package is in-process Go the
// Wails app can bind later. Characters are adults only. No bundled models,
// no cloud calls by default. Kickoff = stubs + contracts, not full art/ML.
package virtualperson
