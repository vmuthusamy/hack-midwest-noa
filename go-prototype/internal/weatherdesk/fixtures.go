package weatherdesk

import "embed"

//go:embed fixtures/*.json
var assets embed.FS
