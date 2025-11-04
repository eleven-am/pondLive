package router

import (
	"net/url"
	"reflect"
	"testing"
)

func TestSetSearch(t *testing.T) {
	input := url.Values{"a": {"1"}}
	got := SetSearch(input, "a", "2", "3")
	want := url.Values{"a": {"2", "3"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SetSearch mismatch: got %#v want %#v", got, want)
	}
}

func TestAddSearch(t *testing.T) {
	input := url.Values{"a": {"1"}}
	got := AddSearch(input, "a", "2")
	want := url.Values{"a": {"1", "2"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AddSearch mismatch: got %#v want %#v", got, want)
	}
}

func TestDelSearch(t *testing.T) {
	input := url.Values{"a": {"1"}, "b": {"x"}}
	got := DelSearch(input, "a")
	want := url.Values{"b": {"x"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DelSearch mismatch: got %#v want %#v", got, want)
	}
}

func TestBuildHref(t *testing.T) {
	input := url.Values{"b": {"y", "x"}, "a": {"1"}}
	href := BuildHref("/p", input, "")
	if href != "/p?a=1&b=x&b=y" {
		t.Fatalf("unexpected href %q", href)
	}
}
