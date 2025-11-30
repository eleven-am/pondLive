package static

import "embed"

//go:embed pondlive.js pondlive-dev.js pondlive-dev.js.map
var Assets embed.FS
