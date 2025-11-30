package static

import "embed"

//go:embed pondlive.js pondlive-dev.js
var Assets embed.FS
