package pkg

import "github.com/eleven-am/pondlive/internal/metatags"

type MetaTag = metatags.MetaTag
type LinkTag = metatags.LinkTag
type ScriptTag = metatags.ScriptTag

func MetaTags(tags ...MetaTag) []Node {
	nodes := metatags.MetaTags(tags...)
	result := make([]Node, len(nodes))
	for i, n := range nodes {
		result[i] = n
	}
	return result
}

func LinkTags(tags ...LinkTag) []Node {
	nodes := metatags.LinkTags(tags...)
	result := make([]Node, len(nodes))
	for i, n := range nodes {
		result[i] = n
	}
	return result
}

func ScriptTags(tags ...ScriptTag) []Node {
	nodes := metatags.ScriptTags(tags...)
	result := make([]Node, len(nodes))
	for i, n := range nodes {
		result[i] = n
	}
	return result
}
