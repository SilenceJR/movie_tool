package organizer

const (
	MovieFolderTemplate = "{{title}} ({{year}})"
	MovieFileTemplate   = "{{title}} ({{year}}) - {{resolution}} {{source}}"

	TVFolderTemplate = "{{title}} ({{year}})/Season {{season}}"
	TVFileTemplate   = "{{title}} - S{{season}}E{{episode}} - {{resolution}} {{source}}"

	AVFolderTemplate = "{{number}} {{title}}"
	AVFileTemplate   = "{{number}} - {{resolution}} {{source}}"
)
