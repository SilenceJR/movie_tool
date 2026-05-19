package organizer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Executor struct {
	Now func() time.Time
	Ops FileOps
}

type FileOps interface {
	Move(source string, target string) error
	Copy(source string, target string) error
	Hardlink(source string, target string) error
	Symlink(source string, target string) error
}

type OSFileOps struct{}

func NewExecutor() Executor {
	return Executor{Now: time.Now, Ops: OSFileOps{}}
}

func (e Executor) Execute(ctx context.Context, plan Plan) Plan {
	now := e.now()
	plan.Status = PlanRunning
	plan.DryRun = false
	plan.UpdatedAt = now

	for index := range plan.Actions {
		if err := ctx.Err(); err != nil {
			plan.Actions[index] = failAction(plan.Actions[index], now, err)
			break
		}
		action := plan.Actions[index]
		if action.Status != ActionPending {
			continue
		}
		if action.SourcePath == "" || action.TargetPath == "" {
			plan.Actions[index] = failAction(action, now, fmt.Errorf("source and target paths are required"))
			continue
		}
		if err := e.executeAction(action); err != nil {
			plan.Actions[index] = failAction(action, now, err)
			continue
		}
		action.Status = ActionSucceeded
		action.Error = ""
		action.ExecutedAt = &now
		plan.Actions[index] = action
	}

	plan.Summary = summarize(plan.Actions)
	plan.Status = executionStatus(plan.Actions)
	plan.UpdatedAt = e.now()
	return plan
}

func (e Executor) executeAction(action Action) error {
	ops := e.Ops
	if ops == nil {
		ops = OSFileOps{}
	}
	switch action.ActionType {
	case ActionMove:
		return ops.Move(action.SourcePath, action.TargetPath)
	case ActionCopy:
		return ops.Copy(action.SourcePath, action.TargetPath)
	case ActionHardlink:
		return ops.Hardlink(action.SourcePath, action.TargetPath)
	case ActionSymlink:
		return ops.Symlink(action.SourcePath, action.TargetPath)
	default:
		return fmt.Errorf("unsupported organizer action %q", action.ActionType)
	}
}

func (e Executor) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

func failAction(action Action, now time.Time, err error) Action {
	action.Status = ActionFailed
	if err != nil {
		action.Error = err.Error()
	}
	action.ExecutedAt = &now
	return action
}

func executionStatus(actions []Action) PlanStatus {
	failed := false
	for _, action := range actions {
		switch action.Status {
		case ActionPending:
			return PlanRunning
		case ActionFailed:
			failed = true
		}
	}
	if failed {
		return PlanFailed
	}
	return PlanSucceeded
}

func (OSFileOps) Move(source string, target string) error {
	if err := ensureParent(target); err != nil {
		return err
	}
	return os.Rename(source, target)
}

func (OSFileOps) Copy(source string, target string) error {
	if err := ensureParent(target); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer output.Close()
	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	return output.Close()
}

func (OSFileOps) Hardlink(source string, target string) error {
	if err := ensureParent(target); err != nil {
		return err
	}
	return os.Link(source, target)
}

func (OSFileOps) Symlink(source string, target string) error {
	if err := ensureParent(target); err != nil {
		return err
	}
	return os.Symlink(source, target)
}

func ensureParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}
