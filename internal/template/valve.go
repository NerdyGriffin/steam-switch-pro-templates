package template

// ValveRelease describes a known Valve-shipped version of a controller template
// file we also manage. Populated manually as new Valve client releases are
// observed. Helps `sspt apply` distinguish "scary, unknown content on disk"
// from "great news, Valve finally shipped this — consider `sspt retire`."
type ValveRelease struct {
	// Hash is the lowercase hex sha256 of the file as Valve shipped it.
	Hash string
	// FirstSeen is a human-readable marker for *when* this hash first appeared
	// (Steam client version, date, or however the maintainer tracked it).
	FirstSeen string
}

// ValveHashes maps a template filename (basename) to every known Valve hash
// for that file. A given filename may have multiple entries because Valve
// has revised official templates between Steam client releases historically.
//
// **Maintainer note:** populate via `sspt scan-valve` (see cmd/sspt/scan_valve.go).
// Run after a Steam client update on a machine that received the update;
// paste the output here. Order does not matter; any match is sufficient.
//
// As of v0.4.0, Valve has not shipped a `controller_switch_pro_gamepad_mouse_gyro.vdf`
// — the absence of which motivated this whole tool. This map is therefore
// intentionally empty for our managed filename. If Valve adds one, add its
// hash and `sspt apply` will surface a friendlier conflict message.
var ValveHashes = map[string][]ValveRelease{
	// "controller_switch_pro_gamepad_mouse_gyro.vdf": {
	//     {Hash: "abc...", FirstSeen: "Steam client 1.0.0.79 (2026-XX)"},
	// },
}

// MatchValveHash returns the matching ValveRelease entry if `diskHash` is a
// known Valve release of `filename`, or nil if not recognized.
func MatchValveHash(filename, diskHash string) *ValveRelease {
	for i := range ValveHashes[filename] {
		if ValveHashes[filename][i].Hash == diskHash {
			return &ValveHashes[filename][i]
		}
	}
	return nil
}
