package template

import (
	"os"
	"path/filepath"
	"testing"
)

// TestApply_ConflictWithValveMatch — when the disk content's hash matches a
// known Valve release, Apply must surface the match in Result.ValveMatch and
// in the human-readable Message.
func TestApply_ConflictWithValveMatch(t *testing.T) {
	inst := fakeInstall(t)
	st := freshState()
	tp := tmpl()

	valveBytes := []byte(`"controller_mappings" { "version" "3" "from" "valve" }`)
	valveHash := HashBytes(valveBytes)
	dst := filepath.Join(inst.TemplatesDir(), tp.Filename)
	if err := os.WriteFile(dst, valveBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	// Inject a known-Valve entry for the test, restore on cleanup.
	prev := ValveHashes[tp.Filename]
	ValveHashes[tp.Filename] = []ValveRelease{
		{Hash: valveHash, FirstSeen: "Steam client X.Y.Z (test)"},
	}
	t.Cleanup(func() { ValveHashes[tp.Filename] = prev })

	res, err := Apply(inst, st, tp, StrategyPreserve, "test")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Outcome != OutcomeConflict {
		t.Fatalf("outcome = %v, want OutcomeConflict", res.Outcome)
	}
	if res.ValveMatch == nil {
		t.Fatal("ValveMatch should be set when disk hash matches a known Valve release")
	}
	if res.ValveMatch.FirstSeen != "Steam client X.Y.Z (test)" {
		t.Fatalf("ValveMatch.FirstSeen = %q, want test marker", res.ValveMatch.FirstSeen)
	}
	if !contains(res.Message, "Valve") || !contains(res.Message, "retire") {
		t.Fatalf("Message should mention Valve + retire suggestion, got: %s", res.Message)
	}
}

// TestApply_ConflictWithoutValveMatch — same flow but disk hash is NOT a
// known Valve release; ValveMatch must be nil and message stays generic.
func TestApply_ConflictWithoutValveMatch(t *testing.T) {
	inst := fakeInstall(t)
	st := freshState()
	tp := tmpl()

	mystery := []byte(`some user edit`)
	dst := filepath.Join(inst.TemplatesDir(), tp.Filename)
	_ = os.WriteFile(dst, mystery, 0o644)

	res, err := Apply(inst, st, tp, StrategyPreserve, "test")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Outcome != OutcomeConflict {
		t.Fatalf("outcome = %v, want OutcomeConflict", res.Outcome)
	}
	if res.ValveMatch != nil {
		t.Fatalf("ValveMatch should be nil for unrecognized content, got %+v", res.ValveMatch)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
