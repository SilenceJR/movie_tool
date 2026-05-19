package strm

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Planner struct{}

func NewPlanner() Planner {
	return Planner{}
}

func (Planner) Build(rule Rule, request GenerateRequest) (Plan, error) {
	if err := validateRule(rule.Name, rule.SourcePrefix, rule.TargetPrefix, rule.OutputPath); err != nil {
		return Plan{}, err
	}
	plan := Plan{
		RuleID:    rule.ID,
		LibraryID: request.LibraryID,
		DryRun:    true,
		Entries:   make([]Entry, 0, len(request.Files)),
	}
	for _, file := range request.Files {
		entry := Entry{
			MediaFileID: file.ID,
			SourcePath:  file.Path,
			Status:      "planned",
		}
		targetURL, outputPath, err := mapPath(rule, file.Path)
		if err != nil {
			entry.Status = "skipped"
			entry.Error = err.Error()
		} else {
			entry.TargetURL = targetURL
			entry.OutputPath = outputPath
			entry.Content = targetURL + "\n"
		}
		plan.Entries = append(plan.Entries, entry)
	}
	plan.Count = len(plan.Entries)
	return plan, nil
}

func (Planner) Validate(ruleInput RuleInput, path string) ValidationResult {
	rule := ruleFromInput(ruleInput)
	if err := validateRule(rule.Name, rule.SourcePrefix, rule.TargetPrefix, rule.OutputPath); err != nil {
		return ValidationResult{Valid: false, Error: err.Error()}
	}
	targetURL, outputPath, err := mapPath(rule, path)
	if err != nil {
		return ValidationResult{Valid: false, Error: err.Error()}
	}
	return ValidationResult{Valid: true, TargetURL: targetURL, OutputPath: outputPath}
}

func mapPath(rule Rule, sourcePath string) (string, string, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return "", "", fmt.Errorf("source path is required")
	}
	cleanSource := filepath.Clean(sourcePath)
	cleanPrefix := filepath.Clean(rule.SourcePrefix)
	if cleanSource != cleanPrefix && !strings.HasPrefix(cleanSource, cleanPrefix+string(filepath.Separator)) {
		return "", "", fmt.Errorf("source path is outside rule source prefix")
	}
	relative, err := filepath.Rel(cleanPrefix, cleanSource)
	if err != nil {
		return "", "", err
	}
	if strings.HasPrefix(relative, "..") {
		return "", "", fmt.Errorf("source path is outside rule source prefix")
	}
	targetURL := joinURL(rule.TargetPrefix, filepath.ToSlash(relative))
	outputRelative := strings.TrimSuffix(filepath.ToSlash(relative), filepath.Ext(relative)) + ".strm"
	outputPath := filepath.Join(rule.OutputPath, filepath.FromSlash(outputRelative))
	return targetURL, outputPath, nil
}

func validateRule(name, sourcePrefix, targetPrefix, outputPath string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("rule name is required")
	}
	if strings.TrimSpace(sourcePrefix) == "" {
		return fmt.Errorf("source prefix is required")
	}
	if strings.TrimSpace(targetPrefix) == "" {
		return fmt.Errorf("target prefix is required")
	}
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("output path is required")
	}
	return nil
}

func joinURL(prefix, relative string) string {
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(relative, "/")
}

func ruleFromInput(input RuleInput) Rule {
	return Rule{
		Name:         input.Name,
		SourcePrefix: input.SourcePrefix,
		TargetPrefix: input.TargetPrefix,
		OutputPath:   input.OutputPath,
		Enabled:      input.Enabled,
	}
}
