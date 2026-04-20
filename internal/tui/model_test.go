package tui

import (
	"testing"
)

func TestMatchMultiKeyword(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input, content string
		want           bool
	}{
		{"", "anything", true},
		{"foo", "FooBar", true},
		{"foo bar", "foo x bar", true},
		{"foo bar", "foo only", false},
		{"  a  b  ", "a and b", true},
	}
	for _, tc := range cases {
		if got := matchMultiKeyword(tc.input, tc.content); got != tc.want {
			t.Fatalf("matchMultiKeyword(%q,%q) = %v, want %v", tc.input, tc.content, got, tc.want)
		}
	}
}

func TestGlobalPaletteFilter(t *testing.T) {
	t.Parallel()
	targets := []string{"alpha beta", "gamma"}
	r := globalPaletteFilter("alp", targets)
	if len(r) != 1 || r[0].Index != 0 {
		t.Fatalf("unexpected ranks: %#v", r)
	}
	r2 := globalPaletteFilter("alp bet", targets)
	if len(r2) != 1 {
		t.Fatalf("multi keyword: %#v", r2)
	}
}

func TestMultiKeywordFilter(t *testing.T) {
	t.Parallel()
	targets := []string{"one two", "three"}
	ranks := multiKeywordFilter("one two", targets)
	if len(ranks) != 1 || ranks[0].Index != 0 {
		t.Fatalf("ranks = %#v", ranks)
	}
}
