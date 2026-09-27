// Package virtualperson is the foundation for Virtual Persons inside SamNPlayer.
//
// Primary product surface: virtual toy props + scene activities.
// MVP: give_toy(dildo) → activity(titjob_dildo) → animation (+ prop sprites).
// Real Sam Neo 2 / Intiface output is an optional parallel channel (ToyHub);
// sync defaults OFF so player.Player remains the sole device writer.
//
// Cherry-picked from draft #264 onto pluginhost H1 (OnFrame wiring). Overlay
// UI and ToyHub ownership flip are follow-ups — do not force-merge #264.
// Characters are adults only. No bundled models, no cloud calls by default.
package virtualperson
