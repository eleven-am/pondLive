package script

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
)

const hashLen = 12

type Asset struct {
	Name string
	Body []byte
	Gzip []byte
	ETag string
	SRI  string
	Hash string
}

func Assemble(core []byte, b *Bundle) (*Asset, error) {
	var code []byte
	if b != nil {
		code = b.Code
	}
	var buf bytes.Buffer
	buf.Grow(len(core) + len(code) + 128)
	buf.Write(core)
	buf.WriteString("\n;(function(define){\n")
	buf.Write(code)
	buf.WriteString("\n})(LiveUIModule.define);\n")
	sum := sha256.Sum256(buf.Bytes())
	hash := hex.EncodeToString(sum[:])[:hashLen]
	buf.WriteString("LiveUIModule.start(\"")
	buf.WriteString(hash)
	buf.WriteString("\");\n")
	body := buf.Bytes()

	var zbuf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&zbuf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(body); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	sri := sha512.Sum384(body)
	return &Asset{
		Name: "pondlive." + hash + ".js",
		Body: body,
		Gzip: zbuf.Bytes(),
		ETag: "\"" + hash + "\"",
		SRI:  "sha384-" + base64.StdEncoding.EncodeToString(sri[:]),
		Hash: hash,
	}, nil
}

type cacheKey struct {
	reg      *Registry
	dev      bool
	features string
	core     [sha256.Size]byte
}

var (
	cacheMu sync.Mutex
	cache   = map[cacheKey]*Asset{}
)

func (r *Registry) Asset(core []byte, opts Options) (*Asset, error) {
	if err := r.Seal(); err != nil {
		return nil, err
	}
	feats := make([]string, len(opts.Features))
	for i, f := range opts.Features {
		feats[i] = string(f)
	}
	sort.Strings(feats)
	key := cacheKey{reg: r, dev: opts.Dev, features: strings.Join(feats, "\x00"), core: sha256.Sum256(core)}
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if a, ok := cache[key]; ok {
		return a, nil
	}
	b, err := r.Bundle(opts)
	if err != nil {
		return nil, err
	}
	a, err := Assemble(core, b)
	if err != nil {
		return nil, err
	}
	cache[key] = a
	return a, nil
}

func BuildAsset(core []byte, opts Options) (*Asset, error) { return Default.Asset(core, opts) }
