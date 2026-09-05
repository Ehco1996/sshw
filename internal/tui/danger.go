package tui

import "github.com/yinheli/sshw"

// dangerousMatch returns the first matched substring and ok=true when cmd
// looks destructive.
func dangerousMatch(cmd string) (string, bool) {
	return sshw.DangerousMatch(cmd)
}

// dangerConfirmPhrase is the exact string the user must type at the
// danger-confirm screen. Long enough to defeat reflex-typing, short
// enough to type at 3am.
const dangerConfirmPhrase = "yes I am sure"
