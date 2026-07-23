package tui

import (
	"strings"
	"testing"
)

func TestRenderBar(t *testing.T) {
	// Zero cases render blank at full width.
	if got := stripStyle(renderBar(0, 0)); got != strings.Repeat(" ", barMaxWidth) {
		t.Fatalf("zero max: %q", got)
	}
	if got := stripStyle(renderBar(0, 100)); got != strings.Repeat(" ", barMaxWidth) {
		t.Fatalf("zero value: %q", got)
	}

	// Full bar: all blocks, no padding.
	got := stripStyle(renderBar(100, 100))
	if got != strings.Repeat("█", barMaxWidth) {
		t.Fatalf("full: %q", got)
	}

	// Tiny non-zero: must produce at least one visible eighth, not blank.
	got = stripStyle(renderBar(1, 10000))
	if strings.TrimSpace(got) == "" {
		t.Fatalf("tiny value rounded to blank: %q", got)
	}
	if len([]rune(got)) != barMaxWidth {
		t.Fatalf("width drift: %d runes", len([]rune(got)))
	}
}

// Strip ANSI SGR sequences so we can assert on the raw glyphs.
func stripStyle(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
