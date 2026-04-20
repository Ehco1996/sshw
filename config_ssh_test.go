package sshw

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSshConfig_missingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-ssh-config")
	t.Setenv("SSHW_SSH_CONFIG_PATH", missing)
	if err := LoadSshConfig(); err != nil {
		t.Fatal(err)
	}
	if GetConfig() != nil {
		t.Fatalf("expected nil config when ~/.ssh/config is absent")
	}
}

func TestLoadSshConfig_readsHost(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	content := "Host mybox\n  HostName 10.0.0.5\n  User deploy\n  Port 2222\n"
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SSHW_SSH_CONFIG_PATH", p)
	if err := LoadSshConfig(); err != nil {
		t.Fatal(err)
	}
	nodes := GetConfig()
	if len(nodes) == 0 {
		t.Fatal("expected at least one node")
	}
	var found bool
	for _, n := range nodes {
		if strings.Contains(n.Host, "10.0.0.5") && n.EffectiveUser() == "deploy" && n.SSHPort() == 2222 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("parsed nodes missing expected host: %#v", nodes)
	}
}
