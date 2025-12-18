package dist

import (
	"embed"
)

//go:embed all:assets
var AssetsDir embed.FS
