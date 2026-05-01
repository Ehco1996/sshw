package tui

import "regexp"

// dangerPatterns flags commands that warrant an extra typed-confirmation
// step. The list is intentionally conservative: false negatives are
// acceptable (this is typo-prevention, not security), but false positives
// train the user to type the override reflexively, which destroys the
// signal. Read this list as "operations that, if accidentally run on N
// hosts at once, are resume-generating events."
var dangerPatterns = []*regexp.Regexp{
	// rm with both -r (or -R) and -f flags, in any order or combination.
	// Matches: rm -rf, rm -fr, rm -Rf, rm -rfv, rm -fRv, rm --recursive --force, etc.
	// Standalone rm -r (no -f) is intentionally NOT flagged: it prompts per file.
	regexp.MustCompile(`(?i)\brm\s+(?:-[a-zA-Z]*(?:r[a-zA-Z]*f|f[a-zA-Z]*r)[a-zA-Z]*|--recursive\s+--force|--force\s+--recursive)`),
	regexp.MustCompile(`(?i)\bmkfs\.\w+\b`),                          // mkfs.ext4 / mkfs.xfs / ...
	regexp.MustCompile(`(?i)\bdd\s+if=`),                             // dd if=...
	regexp.MustCompile(`(?i)\b(shutdown|reboot|poweroff|halt)\b`),    // power state
	regexp.MustCompile(`(?i)\binit\s+[06]\b`),                        // init 0 / init 6
	regexp.MustCompile(`:\s*\(\s*\)\s*\{[^}]*\|\s*:[^}]*\}\s*;\s*:`), // classic fork bomb
	regexp.MustCompile(`(?i)>\s*/dev/[sh]d[a-z]\d*\b`),               // > /dev/sda
	regexp.MustCompile(`(?i)\bchmod\s+(?:-R\s+)?0?00\b`),             // chmod 000 / chmod -R 000
	regexp.MustCompile(`(?i)\bfind\s+/\S*.*\s-delete\b`),             // find / ... -delete
	regexp.MustCompile(`(?i)\bmv\s+\S+\s+/dev/null\b`),               // mv x /dev/null
}

// dangerousMatch returns the first matched substring and ok=true when cmd
// looks destructive. The substring is the exact slice that triggered the
// match (lower- or upper-case as the user typed it), so the UI can
// highlight it inline.
func dangerousMatch(cmd string) (string, bool) {
	for _, re := range dangerPatterns {
		if loc := re.FindStringIndex(cmd); loc != nil {
			return cmd[loc[0]:loc[1]], true
		}
	}
	return "", false
}

// dangerConfirmPhrase is the exact string the user must type at the
// danger-confirm screen. Long enough to defeat reflex-typing, short
// enough to type at 3am.
const dangerConfirmPhrase = "yes I am sure"
