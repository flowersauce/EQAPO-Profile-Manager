package apo

import (
	"bytes"
	"strings"
	"testing"
)

func TestTakeoverPreservesOriginalAndIsIdempotent(t *testing.T) {
	for _, original := range []string{"", "Preamp: -6 dB\n# original\n\n", "\xef\xbb\xbfPreamp: -6 dB\r\nInclude: 中文.txt"} {
		data, err := Takeover([]byte(original))
		if err != nil {
			t.Fatal(err)
		}
		doc, err := Parse(data)
		if err != nil || !doc.Managed || doc.Enabled || doc.Filename != "" {
			t.Fatalf("invalid initial document: %+v, %v", doc, err)
		}
		second, err := Takeover(data)
		if err != nil || !bytes.Equal(data, second) {
			t.Fatalf("reinitialization changed original content: %v", err)
		}
		tail := strings.SplitN(string(data), "# Original config.txt", 2)[1]
		tail = strings.TrimPrefix(strings.TrimPrefix(tail, "\r\n"), "\n")
		var restored strings.Builder
		for _, l := range lines([]byte(tail)) {
			restored.WriteString(strings.TrimPrefix(tail[l.start:l.end], "# "))
		}
		if restored.String() != strings.TrimPrefix(original, "\xef\xbb\xbf") {
			t.Fatalf("original lost: %q", restored.String())
		}
	}
}

func TestSwitchPreservesUnmanagedBytesAndBOM(t *testing.T) {
	data, err := Takeover([]byte("\xef\xbb\xbf# 原始内容\r\nPreamp: -3 dB"))
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := Parse(data)
	on, err := doc.Select("耳机 EQ.txt", true)
	if err != nil {
		t.Fatal(err)
	}
	active, err := Parse(on)
	if err != nil || active.Filename != "耳机 EQ.txt" || !active.Enabled {
		t.Fatalf("selection: %+v %v", active, err)
	}
	off, err := active.Select(active.Filename, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(off, []byte("\xef\xbb\xbf")) || !bytes.Equal(off[len(off)-len(data[doc.end:]):], data[doc.end:]) {
		t.Fatal("unmanaged bytes changed")
	}
	disabled, _ := Parse(off)
	if disabled.Enabled || disabled.Filename != "耳机 EQ.txt" {
		t.Fatal("None lost remembered filename")
	}
}

func TestRejectAmbiguousOrEscapingManagedSections(t *testing.T) {
	for _, input := range []string{
		Begin + "\n# Include:\n", End + "\n", Begin + "\n" + Begin + "\n# Include:\n" + End,
		"Preamp: -3 dB\n" + Begin + "\n# Include:\n" + End,
		Begin + "\nInclude: eqm-profiles\\a.txt\nInclude: eqm-profiles\\b.txt\n" + End,
		Begin + "\nInclude: eqm-profiles\\..\\escape.txt\n" + End,
		Begin + "\nInclude: C:\\escape.txt\n" + End,
		Begin + "\nPreamp: -2 dB\n# Include:\n" + End,
		Begin + "\nInclude:\n" + End,
	} {
		if _, err := Parse([]byte(input)); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestWindowsFilenames(t *testing.T) {
	for _, name := range []string{"None.txt", "CON.txt", "LPT1.foo.txt", "..\\escape.txt", "foo:bar.txt", "foo .txt", ".txt", "a\x1b.txt"} {
		if ValidateFilename(name) == nil {
			t.Errorf("accepted %q", name)
		}
	}
	for _, name := range []string{"耳机 EQ.txt", "Philips SHP9500.TXT", "model.v2.txt"} {
		if err := ValidateFilename(name); err != nil {
			t.Errorf("rejected %q: %v", name, err)
		}
	}
}
