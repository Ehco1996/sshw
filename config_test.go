package sshw

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNode_Connectable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		node *Node
		want bool
	}{
		{"leaf_with_host", &Node{Host: "h.example", Name: "n"}, true},
		{"no_host", &Node{Name: "n"}, false},
		{"host_but_has_children", &Node{Host: "h", Children: []*Node{{Name: "c"}}}, false},
		{"empty_children_leaf", &Node{Host: "h", Children: nil}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.node.Connectable(); got != tc.want {
				t.Fatalf("Connectable() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNode_user_port(t *testing.T) {
	t.Parallel()
	n := &Node{}
	if got := n.user(); got != "root" {
		t.Fatalf("user() empty = %q, want root", got)
	}
	if got := n.EffectiveUser(); got != "root" {
		t.Fatalf("EffectiveUser() empty = %q, want root", got)
	}
	n.User = "alice"
	if got := n.user(); got != "alice" {
		t.Fatalf("user() = %q", got)
	}
	if got := n.EffectiveUser(); got != "alice" {
		t.Fatalf("EffectiveUser() = %q", got)
	}
	n = &Node{}
	if got := n.port(); got != 22 {
		t.Fatalf("port() default = %d", got)
	}
	if got := n.SSHPort(); got != 22 {
		t.Fatalf("SSHPort() default = %d", got)
	}
	for _, p := range []int{0, -1} {
		n.Port = p
		if got := n.port(); got != 22 {
			t.Fatalf("port() for %d = %d, want 22", p, got)
		}
		if got := n.SSHPort(); got != 22 {
			t.Fatalf("SSHPort() for %d = %d, want 22", p, got)
		}
	}
	n.Port = 2222
	if got := n.port(); got != 2222 {
		t.Fatalf("port() = %d", got)
	}
	if got := n.SSHPort(); got != 2222 {
		t.Fatalf("SSHPort() = %d", got)
	}
}

func TestNode_JumpLabel(t *testing.T) {
	t.Parallel()
	n := &Node{}
	if got := n.JumpLabel(); got != "" {
		t.Fatalf("empty jump = %q", got)
	}
	n.Jump = []*Node{{Name: "hop", Host: "10.0.0.1"}}
	if got := n.JumpLabel(); got != "hop" {
		t.Fatalf("JumpLabel = %q, want hop", got)
	}
	n.Jump = []*Node{{Host: "10.0.0.2"}}
	if got := n.JumpLabel(); got != "10.0.0.2" {
		t.Fatalf("JumpLabel = %q", got)
	}
}

func TestFindConnectableByNameOrAlias(t *testing.T) {
	t.Parallel()
	tree := []*Node{
		{
			Name: "g",
			Children: []*Node{
				{Name: "a1", Host: "h1", Alias: "alpha"},
				{Name: "a2", Host: "h2"},
				{Name: "dir", Children: []*Node{{Name: "leaf", Host: "h3"}}},
			},
		},
	}
	tests := []struct {
		token string
		want  int
	}{
		{"a1", 1},
		{"alpha", 1},
		{"leaf", 1},
		{"missing", 0},
		{"g", 0},
	}
	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			got := FindConnectableByNameOrAlias(tree, tt.token)
			if len(got) != tt.want {
				t.Fatalf("len = %d, want %d", len(got), tt.want)
			}
		})
	}
}

func TestLoadConfigBytes_absolutePath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yaml")
	content := []byte("hello")
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfigBytes(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("content = %q, want %q", got, content)
	}
}

func TestLoadConfigBytes_firstMatchWins(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p1 := filepath.Join(dir, "first")
	p2 := filepath.Join(dir, "second")
	_ = os.WriteFile(p1, []byte("one"), 0o600)
	_ = os.WriteFile(p2, []byte("two"), 0o600)
	got, err := LoadConfigBytes(p1, p2)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "one" {
		t.Fatalf("got %q, want one", got)
	}
}

func TestLoadConfigBytes_notFound(t *testing.T) {
	t.Parallel()
	_, err := LoadConfigBytes(filepath.Join(t.TempDir(), "nope-not-exists"))
	if err == nil {
		t.Fatal("expected error")
	}
}
