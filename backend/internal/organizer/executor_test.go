package organizer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeFileOps struct {
	calls []ActionMode
	fail  ActionMode
}

func (o *fakeFileOps) Move(string, string) error     { return o.record(ActionMove) }
func (o *fakeFileOps) Copy(string, string) error     { return o.record(ActionCopy) }
func (o *fakeFileOps) Hardlink(string, string) error { return o.record(ActionHardlink) }
func (o *fakeFileOps) Symlink(string, string) error  { return o.record(ActionSymlink) }

func (o *fakeFileOps) record(mode ActionMode) error {
	o.calls = append(o.calls, mode)
	if o.fail == mode {
		return errors.New("operation failed")
	}
	return nil
}

func TestExecutorExecutesPendingActionsOnly(t *testing.T) {
	ops := &fakeFileOps{}
	executor := Executor{Now: fixedNow, Ops: ops}
	plan := executor.Execute(context.Background(), Plan{
		ID:        "plan-1",
		Status:    PlanReady,
		DryRun:    true,
		CreatedAt: fixedNow(),
		UpdatedAt: fixedNow(),
		Actions: []Action{
			{ID: "a1", ActionType: ActionHardlink, SourcePath: "/downloads/a.mkv", TargetPath: "/library/a.mkv", Status: ActionPending},
			{ID: "a2", ActionType: ActionCopy, SourcePath: "/downloads/b.mkv", TargetPath: "/library/b.mkv", Status: ActionSkipped},
			{ID: "a3", ActionType: ActionMove, SourcePath: "/downloads/c.mkv", TargetPath: "/library/c.mkv", Status: ActionConflict},
		},
	})

	if plan.Status != PlanSucceeded {
		t.Fatalf("expected succeeded plan, got %s", plan.Status)
	}
	if plan.DryRun {
		t.Fatal("expected executed plan to clear dry_run")
	}
	if len(ops.calls) != 1 || ops.calls[0] != ActionHardlink {
		t.Fatalf("expected only hardlink call, got %#v", ops.calls)
	}
	if plan.Actions[0].Status != ActionSucceeded || plan.Actions[0].ExecutedAt == nil {
		t.Fatalf("expected succeeded executed action, got %#v", plan.Actions[0])
	}
	if plan.Actions[1].Status != ActionSkipped || plan.Actions[2].Status != ActionConflict {
		t.Fatalf("expected non-pending statuses preserved, got %#v", plan.Actions)
	}
}

func TestExecutorMarksFailedAction(t *testing.T) {
	ops := &fakeFileOps{fail: ActionMove}
	executor := Executor{Now: fixedNow, Ops: ops}
	plan := executor.Execute(context.Background(), Plan{
		ID:        "plan-1",
		Status:    PlanReady,
		DryRun:    true,
		CreatedAt: fixedNow(),
		UpdatedAt: fixedNow(),
		Actions: []Action{
			{ID: "a1", ActionType: ActionMove, SourcePath: "/downloads/a.mkv", TargetPath: "/library/a.mkv", Status: ActionPending},
		},
	})

	if plan.Status != PlanFailed {
		t.Fatalf("expected failed plan, got %s", plan.Status)
	}
	if plan.Actions[0].Status != ActionFailed || plan.Actions[0].Error != "operation failed" {
		t.Fatalf("expected failed action, got %#v", plan.Actions[0])
	}
}

func TestOSFileOpsCopyAndHardlink(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.mkv")
	copyTarget := filepath.Join(root, "copy", "target.mkv")
	linkTarget := filepath.Join(root, "links", "target.mkv")
	writeTestFile(t, source, "movie")

	ops := OSFileOps{}
	if err := ops.Copy(source, copyTarget); err != nil {
		t.Fatalf("Copy() error = %v", err)
	}
	if err := ops.Hardlink(source, linkTarget); err != nil {
		t.Fatalf("Hardlink() error = %v", err)
	}
	assertFileContent(t, copyTarget, "movie")
	assertFileContent(t, linkTarget, "movie")
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("%s content = %q, want %q", path, string(content), want)
	}
}
