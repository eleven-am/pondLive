package script

import (
	"errors"
	"strings"
	"testing"
)

type strict struct {
	Name string `json:"name"`
	N    int    `json:"n"`
}

type checked struct {
	N int `json:"n"`
}

var errNegative = errors.New("negative")

func (c *checked) Validate() error {
	if c.N < 0 {
		return errNegative
	}
	return nil
}

func TestDecodeIsStrict(t *testing.T) {
	r := NewRegistry()
	d := NewIn[struct{}, strict](r, 0, "export default function(){}")

	got, err := d.Decode([]byte(`{"name":"a","n":3}`))
	if err != nil || got.Name != "a" || got.N != 3 {
		t.Fatalf("valid message: got %+v, %v", got, err)
	}

	for name, raw := range map[string]string{
		"unknown field": `{"name":"a","extra":1}`,
		"wrong type":    `{"name":"a","n":"3"}`,
		"trailing data": `{"name":"a"} {"name":"b"}`,
		"not an object": `[1]`,
		"empty":         ``,
	} {
		if _, err := d.Decode([]byte(raw)); err == nil {
			t.Errorf("%s: %q was accepted", name, raw)
		}
	}

	if _, err := d.DecodeAny([]byte(`{"zzz":1}`)); err == nil {
		t.Error("DecodeAny accepted an unknown field")
	}
}

func TestDecodeValidateIsReachable(t *testing.T) {
	r := NewRegistry()
	d := NewIn[struct{}, checked](r, 0, "export default function(){}")

	ok, err := d.Decode([]byte(`{"n":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Validate(ok); err != nil {
		t.Fatalf("valid message failed validation: %v", err)
	}
	bad, err := d.Decode([]byte(`{"n":-1}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Validate(bad); !errors.Is(err, errNegative) {
		t.Fatalf("Validate not reached: %v", err)
	}
	var e Entry = d
	msg, err := e.DecodeAny([]byte(`{"n":-5}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ValidateAny(msg); !errors.Is(err, errNegative) {
		t.Fatalf("ValidateAny not reached: %v", err)
	}
	if err := e.ValidateAny("wrong type"); err == nil {
		t.Fatal("ValidateAny accepted a foreign message type")
	}
}

func TestEncode(t *testing.T) {
	r := NewRegistry()
	d := NewIn[strict, struct{}](r, 0, "export default function(){}")
	b, err := d.Encode(strict{Name: "x", N: 2})
	if err != nil || string(b) != `{"name":"x","n":2}` {
		t.Fatalf("Encode: %s, %v", b, err)
	}
}

func TestDefaultsAndOptions(t *testing.T) {
	r := NewRegistry()
	d := NewIn[struct{}, struct{}](r, 0, "a")
	l := d.Limits()
	if l.MaxMessage != 16*1024 || l.Rate != 60 || l.Burst != 120 {
		t.Fatalf("defaults: %+v", l)
	}
	d2 := NewIn[struct{}, struct{}](r, 0, "b", MaxMessage(100), Rate(5, 7))
	l = d2.Limits()
	if l.MaxMessage != 100 || l.Rate != 5 || l.Burst != 7 {
		t.Fatalf("options: %+v", l)
	}
	for name, fn := range map[string]func(){
		"MaxMessage zero": func() { MaxMessage(0) },
		"Rate zero":       func() { Rate(0, 1) },
		"Burst zero":      func() { Rate(1, 0) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", name)
				}
			}()
			fn()
		}()
	}
}

func TestKeyIsContentAddressed(t *testing.T) {
	r := NewRegistry()
	a := NewIn[struct{}, struct{}](r, 0, "one")
	b := NewIn[struct{}, struct{}](r, 0, "two")
	if a.Key() == b.Key() || len(a.Key()) != 12 {
		t.Fatalf("keys %q %q", a.Key(), b.Key())
	}
	if deriveKey("inline", "one") != a.Key() {
		t.Fatal("key is not sha256(inline || 0x00 || source)")
	}
	if deriveKey("x.ts", "one") == a.Key() {
		t.Fatal("entry path is not part of the key")
	}
}

func TestIdenticalSourceSharesOneDefinition(t *testing.T) {
	r := NewRegistry()
	a := NewIn[struct{}, struct{}](r, 0, "same")
	b := NewIn[struct{}, struct{}](r, 0, "same")
	if a != b {
		t.Fatal("identical source produced two definitions")
	}
	if err := r.Seal(); err != nil {
		t.Fatal(err)
	}
	if len(r.Entries()) != 1 {
		t.Fatalf("entries: %d", len(r.Entries()))
	}
}

func TestIdenticalSourceWithDifferentTypesFails(t *testing.T) {
	r := NewRegistry()
	NewIn[struct{}, strict](r, 0, "same")
	NewIn[strict, struct{}](r, 0, "same")
	err := r.Seal()
	if err == nil || !strings.Contains(err.Error(), "different message types") {
		t.Fatalf("got %v", err)
	}
}

func TestCallerLocationNamesTheDeclaration(t *testing.T) {
	r := NewRegistry()
	d := NewIn[struct{}, struct{}](r, 0, "x")
	if !strings.HasPrefix(d.Location(), "def_test.go:") {
		t.Fatalf("location %q", d.Location())
	}
}
