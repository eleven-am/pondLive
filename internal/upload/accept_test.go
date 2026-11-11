package upload

import (
	"testing"

	"github.com/gabriel-vasile/mimetype"
)

var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func TestAcceptsDeclared(t *testing.T) {
	tests := []struct {
		name   string
		accept []string
		file   string
		typ    string
		want   bool
	}{
		{"empty accepts all", nil, "a.bin", "", true},
		{"exact type", []string{"image/png"}, "a.png", "image/png", true},
		{"exact type mismatch", []string{"image/png"}, "a.jpg", "image/jpeg", false},
		{"exact type ignores params and case", []string{"Text/Plain"}, "a.txt", "text/plain; charset=utf-8", true},
		{"wildcard", []string{"image/*"}, "a.webp", "image/webp", true},
		{"wildcard mismatch", []string{"image/*"}, "a.pdf", "application/pdf", false},
		{"wildcard needs a type", []string{"image/*"}, "a.png", "", false},
		{"extension", []string{".png"}, "photo.PNG", "", true},
		{"extension mismatch", []string{".png"}, "photo.jpg", "image/png", false},
		{"multi-dot extension", []string{".tar.gz"}, "x.tar.gz", "", true},
		{"any", []string{"*/*"}, "x", "", true},
		{"blank entries ignored", []string{" ", ""}, "x.png", "image/png", false},
		{"one of several", []string{"application/pdf", ".png"}, "x.png", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AcceptsDeclared(tt.accept, tt.file, tt.typ); got != tt.want {
				t.Fatalf("AcceptsDeclared(%v, %q, %q) = %v, want %v", tt.accept, tt.file, tt.typ, got, tt.want)
			}
		})
	}
}

func TestAcceptsContent(t *testing.T) {
	png := mimetype.Detect(pngBytes)
	text := mimetype.Detect([]byte("just some text"))
	jsonDoc := mimetype.Detect([]byte(`{"a": 1}`))
	if png.String() != "image/png" {
		t.Fatalf("fixture is not detected as png: %s", png)
	}

	tests := []struct {
		name   string
		accept []string
		file   string
		m      *mimetype.MIME
		want   bool
	}{
		{"empty accepts all", nil, "a", text, true},
		{"exact type", []string{"image/png"}, "a.png", png, true},
		{"exact type rejects text", []string{"image/png"}, "a.png", text, false},
		{"wildcard", []string{"image/*"}, "a.png", png, true},
		{"wildcard rejects text", []string{"image/*"}, "a.png", text, false},
		{"application wildcard does not match images", []string{"application/*"}, "a.png", png, false},
		{"extension", []string{".png"}, "a.png", png, true},
		{"extension rejects text posing as png", []string{".png"}, "a.png", text, false},
		{"extension is enforced by content, not by the client name", []string{".png"}, "a.txt", png, true},
		{"markdown extension accepts plain text content", []string{".md"}, "readme.md", mimetype.Detect([]byte("# Title\n\nhello world\n")), true},
		{"csv extension accepts csv content", []string{".csv"}, "a.csv", mimetype.Detect([]byte("a,b,c\n1,2,3\n4,5,6\n")), true},
		{"text/markdown target accepts sniffed plain text", []string{"text/markdown"}, "a.md", text, true},
		{"text wildcard accepts sniffed plain text", []string{"text/*"}, "a", text, true},
		{"text extension rejects a binary", []string{".md"}, "a.md", png, false},
		{"octet-stream content satisfies only its own family", []string{"image/png"}, "a.png", mimetype.Detect([]byte{0x01, 0xfe, 0x00, 0x01, 0xfe, 0x00}), false},
		{"unknown extension cannot be verified", []string{".zzqq"}, "a.zzqq", text, false},
		{"pdf extension rejects text", []string{".pdf"}, "a.pdf", text, false},
		{"extension alias", []string{".jpeg"}, "a.jpeg", mimetype.Detect([]byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01")), true},
		{"text type accepts json via parent", []string{"text/plain"}, "a.json", jsonDoc, true},
		{"any", []string{"*/*"}, "a", text, true},
		{"nil detection rejected", []string{"image/png"}, "a.png", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AcceptsContent(tt.accept, tt.file, tt.m); got != tt.want {
				t.Fatalf("AcceptsContent(%v, %q, %v) = %v, want %v", tt.accept, tt.file, tt.m, got, tt.want)
			}
		})
	}
}

func TestAcceptsUploadRejectsIncompatibleDeclaredTypeEarly(t *testing.T) {
	tests := []struct {
		name   string
		accept []string
		file   string
		typ    string
		want   bool
	}{
		{"extension with matching declared type", []string{".png"}, "a.png", "image/png", true},
		{"extension with empty declared type", []string{".png"}, "a.png", "", true},
		{"extension with generic declared type", []string{".png"}, "a.png", "application/octet-stream", true},
		{"extension with incompatible declared type", []string{".png"}, "a.png", "application/pdf", false},
		{"markdown with text/plain declared", []string{".md"}, "a.md", "text/plain", true},
		{"markdown with a binary declared", []string{".md"}, "a.md", "image/png", false},
		{"jpeg alias declared", []string{".jpeg"}, "a.jpeg", "image/jpeg", true},
		{"unknown extension is rejected at create", []string{".zzqq"}, "a.zzqq", "", false},
		{"type pattern unchanged", []string{"image/*"}, "a.png", "image/png", true},
		{"other pattern still rescues", []string{".png", "application/pdf"}, "a.png", "application/pdf", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AcceptsUpload(tt.accept, tt.file, tt.typ); got != tt.want {
				t.Fatalf("AcceptsUpload(%v, %q, %q) = %v, want %v", tt.accept, tt.file, tt.typ, got, tt.want)
			}
		})
	}
}

func TestUnknownExtensions(t *testing.T) {
	got := UnknownExtensions([]string{".MOV", ".png", "image/*", ".mkv", " .Sql ", ".md", ""})
	want := []string{".mov", ".mkv", ".sql"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if len(UnknownExtensions(nil)) != 0 {
		t.Fatal("nil accept has no unknown extensions")
	}
}
