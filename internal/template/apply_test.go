package template

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/state"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/steam"
)

// fakeInstall builds a *steam.Install rooted at a temp dir with the
// controller_base/templates/ subtree present.
func fakeInstall(t *testing.T) *steam.Install {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "controller_base", "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &steam.Install{Root: root}
}

func tmpl() Template {
	body := []byte(`"controller_mappings" { "version" "3" }`)
	return Template{Filename: "test_template.vdf", Content: body, Hash: HashBytes(body)}
}

func freshState() *state.State {
	return &state.State{Templates: map[string]TemplateState{}}
}

// TemplateState alias so callers don't pull in state.TemplateState directly here.
type TemplateState = state.TemplateState

func TestApply_AbsentInstalls(t *testing.T) {
	inst := fakeInstall(t)
	st := freshState()
	tp := tmpl()

	res, err := Apply(inst, st, tp, StrategyPreserve, "test")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Outcome != OutcomeInstalled {
		t.Fatalf("outcome = %v, want OutcomeInstalled", res.Outcome)
	}

	on, err := os.ReadFile(filepath.Join(inst.TemplatesDir(), tp.Filename))
	if err != nil {
		t.Fatalf("read installed file: %v", err)
	}
	if HashBytes(on) != tp.Hash {
		t.Fatal("installed file hash mismatch")
	}
	if got := st.Templates[tp.Filename]; got.InstalledHash != tp.Hash {
		t.Fatalf("state hash = %s, want %s", got.InstalledHash, tp.Hash)
	}
}

func TestApply_AlreadyCurrentIsNoOp(t *testing.T) {
	inst := fakeInstall(t)
	st := freshState()
	tp := tmpl()

	if err := os.WriteFile(filepath.Join(inst.TemplatesDir(), tp.Filename), tp.Content, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Apply(inst, st, tp, StrategyPreserve, "test")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Outcome != OutcomeAlreadyCurrent {
		t.Fatalf("outcome = %v, want OutcomeAlreadyCurrent", res.Outcome)
	}
}

func TestApply_UpgradesFromOlderEmbed(t *testing.T) {
	inst := fakeInstall(t)
	st := freshState()

	old := []byte(`"controller_mappings" { "version" "2" }`)
	oldHash := HashBytes(old)
	new_ := tmpl()

	// Simulate: previous run installed `old`. Now embedded copy is `new_`.
	dst := filepath.Join(inst.TemplatesDir(), new_.Filename)
	if err := os.WriteFile(dst, old, 0o644); err != nil {
		t.Fatal(err)
	}
	st.Templates[new_.Filename] = TemplateState{
		InstalledHash:    oldHash,
		InstalledVersion: "0.0.1",
		InstalledAt:      time.Now().Add(-time.Hour),
	}

	res, err := Apply(inst, st, new_, StrategyPreserve, "test")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Outcome != OutcomeUpgraded {
		t.Fatalf("outcome = %v, want OutcomeUpgraded", res.Outcome)
	}
	on, _ := os.ReadFile(dst)
	if HashBytes(on) != new_.Hash {
		t.Fatal("disk content not upgraded")
	}
}

func TestApply_PreservesUnknownContent(t *testing.T) {
	inst := fakeInstall(t)
	st := freshState()
	tp := tmpl()

	mystery := []byte(`"controller_mappings" { "version" "3" "manual_edit" "yes" }`)
	dst := filepath.Join(inst.TemplatesDir(), tp.Filename)
	if err := os.WriteFile(dst, mystery, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Apply(inst, st, tp, StrategyPreserve, "test")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Outcome != OutcomeConflict {
		t.Fatalf("outcome = %v, want OutcomeConflict", res.Outcome)
	}
	on, _ := os.ReadFile(dst)
	if HashBytes(on) != HashBytes(mystery) {
		t.Fatal("file was modified despite preserve strategy")
	}
	if st.Templates[tp.Filename].ConflictSeenAt == nil {
		t.Fatal("conflict timestamp not recorded")
	}
}

func TestApply_ForceOverwritesUnknown(t *testing.T) {
	inst := fakeInstall(t)
	st := freshState()
	tp := tmpl()

	mystery := []byte(`mystery content`)
	dst := filepath.Join(inst.TemplatesDir(), tp.Filename)
	_ = os.WriteFile(dst, mystery, 0o644)

	res, err := Apply(inst, st, tp, StrategyForce, "test")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Outcome != OutcomeForced {
		t.Fatalf("outcome = %v, want OutcomeForced", res.Outcome)
	}
	on, _ := os.ReadFile(dst)
	if HashBytes(on) != tp.Hash {
		t.Fatal("force did not overwrite to embedded content")
	}
}

func TestApply_AcceptExistingAdoptsHash(t *testing.T) {
	inst := fakeInstall(t)
	st := freshState()
	tp := tmpl()

	mystery := []byte(`from valve, hypothetically`)
	dst := filepath.Join(inst.TemplatesDir(), tp.Filename)
	_ = os.WriteFile(dst, mystery, 0o644)

	res, err := Apply(inst, st, tp, StrategyAcceptExisting, "test")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Outcome != OutcomeAccepted {
		t.Fatalf("outcome = %v, want OutcomeAccepted", res.Outcome)
	}
	on, _ := os.ReadFile(dst)
	if HashBytes(on) != HashBytes(mystery) {
		t.Fatal("accept should not modify on-disk content")
	}
	if st.Templates[tp.Filename].InstalledHash != HashBytes(mystery) {
		t.Fatal("state did not adopt disk hash")
	}
}

func TestApply_RecognizesOurOwnEvenWithoutState(t *testing.T) {
	// If state is lost (user wiped %LOCALAPPDATA%) but the disk file still
	// matches the embedded copy, apply should treat it as ours, not as a
	// conflict.
	inst := fakeInstall(t)
	st := freshState()
	tp := tmpl()

	dst := filepath.Join(inst.TemplatesDir(), tp.Filename)
	_ = os.WriteFile(dst, tp.Content, 0o644)

	res, err := Apply(inst, st, tp, StrategyPreserve, "test")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Outcome != OutcomeAlreadyCurrent {
		t.Fatalf("outcome = %v, want OutcomeAlreadyCurrent", res.Outcome)
	}
	if st.Templates[tp.Filename].InstalledHash != tp.Hash {
		t.Fatal("state was not reconstructed from matching disk content")
	}
}
