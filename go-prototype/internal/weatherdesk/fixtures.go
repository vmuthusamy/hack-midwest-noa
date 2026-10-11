package weatherdesk

import "embed"

//go:embed fixtures/*.json case-recordings/*.json
var assets embed.FS
