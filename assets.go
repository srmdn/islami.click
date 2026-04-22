package islamiclick

import "embed"

//go:embed templates
var TemplateFS embed.FS

//go:embed content
var ContentFS embed.FS

//go:embed migrations
var MigrationFS embed.FS
