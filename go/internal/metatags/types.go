package metatags

import (
	"crypto/sha256"
	"fmt"
)

type Meta struct {
	Title       string
	Description string
	Meta        []MetaTag
	Links       []LinkTag
	Scripts     []ScriptTag
}

type metaEntry struct {
	meta        *Meta
	depth       int
	componentID string
}

type scriptEntry struct {
	script ScriptTag
	depth  int
}

func inlineScriptKey(componentID string, depth, idx int, script ScriptTag) string {
	data := script.Inner + "|" + script.Nonce + "|" + script.Type + "|" + script.Src
	return fmt.Sprintf("inline:%s:%d:%d:%x", componentID, depth, idx, sha256.Sum256([]byte(data)))
}

var defaultMeta = &Meta{
	Title:       "PondLive Application",
	Description: "A PondLive application",
}
