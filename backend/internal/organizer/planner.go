package organizer

import (
	"bytes"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	MediaTypeMovie = "movie"
	MediaTypeTV    = "tv"
	MediaTypeAV    = "av"
)

var ErrRuleDisabled = errors.New("organizer rule is disabled")

type MediaInfo struct {
	ID            string `json:"id"`
	LibraryID     string `json:"library_id"`
	MediaType     string `json:"media_type"`
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title"`
	DisplayTitle  string `json:"display_title"`
	Year          int    `json:"year"`
	Number        string `json:"number"`
}

type VersionInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Resolution   string `json:"resolution"`
	Source       string `json:"source"`
	VideoCodec   string `json:"video_codec"`
	AudioCodec   string `json:"audio_codec"`
	HDRFormat    string `json:"hdr_format"`
	Edition      string `json:"edition"`
	ReleaseGroup string `json:"release_group"`
	IsDefault    bool   `json:"is_default"`
}

type FileInfo struct {
	ID        string `json:"id"`
	MediaID   string `json:"media_id"`
	VersionID string `json:"version_id"`
	Path      string `json:"path"`
	FileName  string `json:"file_name"`
	Extension string `json:"extension"`
	Season    int    `json:"season"`
	Episode   int    `json:"episode"`
	Number    string `json:"number"`
}

type PlanRequest struct {
	MediaID   string        `json:"media_id"`
	LibraryID string        `json:"library_id"`
	Media     MediaInfo     `json:"media"`
	Versions  []VersionInfo `json:"versions"`
	Files     []FileInfo    `json:"files"`
	RuleID    string        `json:"rule_id"`
	Rule      Rule          `json:"rule"`
}

type Planner struct {
	Now func() time.Time
}

func NewPlanner() Planner {
	return Planner{Now: time.Now}
}

func (p Planner) Build(request PlanRequest) (Plan, error) {
	if !request.Rule.Enabled {
		return Plan{}, ErrRuleDisabled
	}

	now := p.now()
	rule := withDefaultRuleTemplates(request.Rule, request.Media.MediaType)
	versions := indexVersions(request.Versions)
	seenTargets := make(map[string]string)

	plan := Plan{
		ID:        "plan-" + stablePart(request.Media.ID, "media"),
		LibraryID: firstNonEmpty(rule.LibraryID, request.Media.LibraryID),
		Status:    PlanReady,
		DryRun:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	for index, file := range request.Files {
		version := versions[file.VersionID]
		values := templateValues(request.Media, version, file)
		folder, err := renderTemplate(rule.FolderTemplate, values)
		if err != nil {
			return Plan{}, err
		}
		name, err := renderTemplate(rule.FileTemplate, values)
		if err != nil {
			return Plan{}, err
		}

		targetPath := filepath.Join(rule.TargetRoot, cleanRelativePath(folder), cleanFileName(name)+fileExtension(file))
		action := Action{
			ID:          "action-" + strconv.Itoa(index+1),
			PlanID:      plan.ID,
			MediaID:     firstNonEmpty(file.MediaID, request.Media.ID),
			MediaFileID: file.ID,
			ActionType:  defaultActionMode(rule.ActionMode),
			SourcePath:  file.Path,
			TargetPath:  targetPath,
			Status:      ActionPending,
			CreatedAt:   now,
		}
		if previousSource, ok := seenTargets[targetPath]; ok && previousSource != file.Path {
			action.Status = ActionConflict
			action.ConflictReason = "duplicate target path in plan"
		}
		seenTargets[targetPath] = file.Path
		plan.Actions = append(plan.Actions, action)
	}

	plan.Summary = summarize(plan.Actions)
	return plan, nil
}

func (p Planner) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

func withDefaultRuleTemplates(rule Rule, mediaType string) Rule {
	if rule.FolderTemplate != "" && rule.FileTemplate != "" {
		return rule
	}

	switch mediaType {
	case MediaTypeTV:
		if rule.FolderTemplate == "" {
			rule.FolderTemplate = TVFolderTemplate
		}
		if rule.FileTemplate == "" {
			rule.FileTemplate = TVFileTemplate
		}
	case MediaTypeAV:
		if rule.FolderTemplate == "" {
			rule.FolderTemplate = AVFolderTemplate
		}
		if rule.FileTemplate == "" {
			rule.FileTemplate = AVFileTemplate
		}
	default:
		if rule.FolderTemplate == "" {
			rule.FolderTemplate = MovieFolderTemplate
		}
		if rule.FileTemplate == "" {
			rule.FileTemplate = MovieFileTemplate
		}
	}
	return rule
}

func indexVersions(versions []VersionInfo) map[string]VersionInfo {
	result := make(map[string]VersionInfo, len(versions))
	for _, version := range versions {
		result[version.ID] = version
	}
	return result
}

func templateValues(media MediaInfo, version VersionInfo, file FileInfo) map[string]string {
	title := firstNonEmpty(media.DisplayTitle, media.Title, media.OriginalTitle)
	number := firstNonEmpty(file.Number, media.Number)
	values := map[string]string{
		"title":          title,
		"original_title": media.OriginalTitle,
		"display_title":  media.DisplayTitle,
		"year":           intString(media.Year),
		"number":         number,
		"season":         twoDigit(file.Season),
		"episode":        twoDigit(file.Episode),
		"version":        version.Name,
		"resolution":     version.Resolution,
		"source":         version.Source,
		"video_codec":    version.VideoCodec,
		"audio_codec":    version.AudioCodec,
		"hdr_format":     version.HDRFormat,
		"edition":        version.Edition,
		"release_group":  version.ReleaseGroup,
	}
	return values
}

func renderTemplate(pattern string, values map[string]string) (string, error) {
	funcs := template.FuncMap{}
	for key, value := range values {
		value := value
		funcs[key] = func() string {
			return value
		}
	}
	tmpl, err := template.New("organizer").Funcs(funcs).Option("missingkey=zero").Parse(pattern)
	if err != nil {
		return "", err
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, values); err != nil {
		return "", err
	}
	return strings.Join(strings.Fields(rendered.String()), " "), nil
}

func cleanRelativePath(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for index, part := range parts {
		parts[index] = cleanFileName(part)
	}
	return filepath.Join(parts...)
}

func cleanFileName(value string) string {
	replacer := strings.NewReplacer("/", " ", "\\", " ", ":", " -", "*", " ", "?", " ", "\"", "'", "<", " ", ">", " ", "|", " ")
	return strings.TrimSpace(strings.Join(strings.Fields(replacer.Replace(value)), " "))
}

func fileExtension(file FileInfo) string {
	if file.Extension != "" {
		if strings.HasPrefix(file.Extension, ".") {
			return strings.ToLower(file.Extension)
		}
		return "." + strings.ToLower(file.Extension)
	}
	return strings.ToLower(filepath.Ext(file.Path))
}

func summarize(actions []Action) Summary {
	summary := Summary{TotalActions: len(actions)}
	for _, action := range actions {
		switch action.ActionType {
		case ActionMove:
			summary.MoveCount++
		case ActionCopy:
			summary.CopyCount++
		case ActionHardlink, ActionSymlink:
			summary.LinkCount++
		}
		switch action.Status {
		case ActionConflict:
			summary.ConflictCount++
		case ActionSkipped:
			summary.SkipCount++
		}
	}
	return summary
}

func defaultActionMode(mode ActionMode) ActionMode {
	if mode == "" {
		return ActionMove
	}
	return mode
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stablePart(value string, fallback string) string {
	if value != "" {
		return cleanFileName(value)
	}
	return fallback
}

func intString(value int) string {
	if value == 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func twoDigit(value int) string {
	if value <= 0 {
		return ""
	}
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
