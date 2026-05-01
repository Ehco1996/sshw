package tui

import "strings"

// stripAnsi removes CSI escape sequences (ESC [ ... letter) from s.
// Used by both bucket-key normalization (so colored stdout coalesces with
// uncolored equivalents) and tests that assert on rendered substrings.
// Not a fully general ANSI parser — handles the SGR / cursor-control
// sequences typical of `ls --color`, `git status`, etc.
func stripAnsi(s string) string {
	if !strings.ContainsRune(s, 0x1b) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	inEsc := false
	for _, r := range s {
		if r == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			// CSI sequences end on a letter @-~ (0x40-0x7e). For SGR that's
			// 'm'; cursor moves use 'A'..'H' etc. Stop on any of those.
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '@' || r == '~' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// normalizeOutput trims trailing whitespace per line and drops trailing empty
// lines, then strips ANSI. Used to compute bucket keys that ignore cosmetic
// diffs while keeping semantic content intact.
func normalizeOutput(s string) string {
	stripped := stripAnsi(s)
	lines := strings.Split(stripped, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	// Trim trailing empty lines.
	end := len(lines)
	for end > 0 && lines[end-1] == "" {
		end--
	}
	return strings.Join(lines[:end], "\n")
}
