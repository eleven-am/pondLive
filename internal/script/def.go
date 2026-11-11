package script

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"runtime"
)

const (
	DefaultMaxMessage = 16 * 1024
	DefaultRate       = 60
	DefaultBurst      = 120
	inlineEntry       = "inline"
	keyLen            = 12
)

type Validator interface {
	Validate() error
}

type Opt func(*config)

type config struct {
	limits  Limits
	feature Feature
}

func MaxMessage(bytes int) Opt {
	if bytes <= 0 {
		panic(fmt.Sprintf("script: MaxMessage must be positive, got %d", bytes))
	}
	return func(c *config) { c.limits.MaxMessage = bytes }
}

func Rate(perSecond float64, burst int) Opt {
	if perSecond <= 0 || burst <= 0 {
		panic(fmt.Sprintf("script: Rate needs a positive rate and burst, got %v and %d", perSecond, burst))
	}
	return func(c *config) {
		c.limits.Rate = perSecond
		c.limits.Burst = burst
	}
}

func OnlyWith(f Feature) Opt {
	return func(c *config) { c.feature = f }
}

type Def[Down, Up any] struct {
	key      string
	source   string
	entry    string
	fsys     fs.FS
	location string
	limits   Limits
	feature  Feature
}

func New[Down, Up any](source string, opts ...Opt) *Def[Down, Up] {
	return NewIn[Down, Up](Default, 1, source, opts...)
}

func NewFS[Down, Up any](fsys fs.FS, entry string, opts ...Opt) *Def[Down, Up] {
	return NewFSIn[Down, Up](Default, 1, fsys, entry, opts...)
}

func NewIn[Down, Up any](r *Registry, skip int, source string, opts ...Opt) *Def[Down, Up] {
	d := newDef[Down, Up](skip, inlineEntry, source, nil, opts)
	return register(r, d)
}

func NewFSIn[Down, Up any](r *Registry, skip int, fsys fs.FS, entry string, opts ...Opt) *Def[Down, Up] {
	d := newDef[Down, Up](skip, entry, "", fsys, opts)
	if !fs.ValidPath(entry) {
		r.fail(fmt.Errorf("%s: script entry %q is not a valid FS path", d.location, entry))
		return d
	}
	data, err := fs.ReadFile(fsys, entry)
	if err != nil {
		r.fail(fmt.Errorf("%s: reading script entry %q: %w", d.location, entry, err))
		return d
	}
	d.source = string(data)
	d.key = deriveKey(entry, d.source)
	return register(r, d)
}

func newDef[Down, Up any](skip int, entry, source string, fsys fs.FS, opts []Opt) *Def[Down, Up] {
	c := config{limits: Limits{MaxMessage: DefaultMaxMessage, Rate: DefaultRate, Burst: DefaultBurst}}
	for _, o := range opts {
		o(&c)
	}
	return &Def[Down, Up]{
		key:      deriveKey(entry, source),
		source:   source,
		entry:    entry,
		fsys:     fsys,
		location: callerLocation(skip + 3),
		limits:   c.limits,
		feature:  c.feature,
	}
}

func register[Down, Up any](r *Registry, d *Def[Down, Up]) *Def[Down, Up] {
	got := r.Register(d)
	if same, ok := got.(*Def[Down, Up]); ok {
		return same
	}
	r.fail(fmt.Errorf("%s: script %s is already declared with different message types at %s", d.location, d.key, got.Location()))
	return d
}

func deriveKey(entry, source string) string {
	h := sha256.New()
	h.Write([]byte(entry))
	h.Write([]byte{0})
	h.Write([]byte(source))
	return hex.EncodeToString(h.Sum(nil))[:keyLen]
}

func callerLocation(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown:0"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

func (d *Def[Down, Up]) Key() string      { return d.key }
func (d *Def[Down, Up]) Source() string   { return d.source }
func (d *Def[Down, Up]) Entry() string    { return d.entry }
func (d *Def[Down, Up]) FS() fs.FS        { return d.fsys }
func (d *Def[Down, Up]) Location() string { return d.location }
func (d *Def[Down, Up]) Limits() Limits   { return d.limits }
func (d *Def[Down, Up]) Feature() Feature { return d.feature }

func (d *Def[Down, Up]) Decode(raw []byte) (Up, error) {
	var up Up
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&up); err != nil {
		var zero Up
		return zero, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		var zero Up
		return zero, errors.New("trailing data after message")
	}
	return up, nil
}

func (d *Def[Down, Up]) Validate(up Up) error {
	if v, ok := any(&up).(Validator); ok {
		return v.Validate()
	}
	return nil
}

func (d *Def[Down, Up]) Encode(msg Down) ([]byte, error) {
	return json.Marshal(msg)
}

func (d *Def[Down, Up]) DecodeAny(raw []byte) (any, error) {
	up, err := d.Decode(raw)
	if err != nil {
		return nil, err
	}
	return up, nil
}

func (d *Def[Down, Up]) ValidateAny(msg any) error {
	up, ok := msg.(Up)
	if !ok {
		return fmt.Errorf("script %s: message is %T, want %T", d.key, msg, *new(Up))
	}
	return d.Validate(up)
}
