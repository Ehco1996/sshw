package tui

import "testing"

func TestDangerousMatch(t *testing.T) {
	t.Parallel()

	positive := []struct {
		cmd, want string
	}{
		{"rm -rf /var/log/old", "rm -rf"},
		{"RM -RF /tmp", "RM -RF"},
		{"sudo rm -fr /opt/legacy", "rm -fr"},
		{"rm -Rf x", "rm -Rf"},
		{"rm -rfv x", "rm -rfv"},
		{"rm --recursive --force x", "rm --recursive --force"},
		{"dd if=/dev/zero of=/dev/sda bs=1M", "dd if="},
		{"mkfs.ext4 /dev/sdb1", "mkfs.ext4"},
		{"shutdown -h now", "shutdown"},
		{"reboot", "reboot"},
		{"init 0", "init 0"},
		{":(){ :|:& };:", ":(){ :|:& };:"},
		{"cat /tmp/junk > /dev/sda", "> /dev/sda"},
		{"chmod -R 000 /opt", "chmod -R 000"},
		{"find / -name foo -delete", "find / -name foo -delete"},
		{"mv /etc/important /dev/null", "mv /etc/important /dev/null"},
	}
	for _, tc := range positive {
		got, ok := dangerousMatch(tc.cmd)
		if !ok {
			t.Errorf("expected match for %q, got none", tc.cmd)
			continue
		}
		if got != tc.want {
			t.Errorf("dangerousMatch(%q) = %q, want %q", tc.cmd, got, tc.want)
		}
	}

	negative := []string{
		"",
		"ls -la",
		"rm file.txt",
		"rm -f file.txt", // -f without -r is allowed (single file deletion)
		"cat /etc/shadow",
		// Note: "echo rm -rf" is a known false positive — we don't try to
		// parse the shell, so any string containing "rm -rf" trips the guard.
		// Users can press esc and clean up.
		"git status",
		"systemctl restart nginx",
		"ps aux | grep dd",
		"docker ps",
		"uptime",
		"df -h",
	}
	for _, cmd := range negative {
		if got, ok := dangerousMatch(cmd); ok {
			t.Errorf("expected no match for %q, got %q", cmd, got)
		}
	}
}
