package sshw

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestApplyEditForm_createLeaf(t *testing.T) {
	t.Parallel()
	n, err := ApplyEditForm(EditFormValues{
		Name: "web", Host: "10.0.0.1", User: "deploy", Port: "2222", Alias: "w",
		KeyPath: "~/.ssh/id_ed25519", Passphrase: "sekret", Password: "hunter2",
	}, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if n.Name != "web" || n.Host != "10.0.0.1" || n.User != "deploy" || n.Port != 2222 || n.Alias != "w" {
		t.Fatalf("unexpected node: %#v", n)
	}
	if n.KeyPath != "~/.ssh/id_ed25519" || n.Passphrase != "sekret" || n.Password != "hunter2" {
		t.Fatalf("auth fields: %#v", n)
	}
}

func TestApplyEditForm_createGroup(t *testing.T) {
	t.Parallel()
	n, err := ApplyEditForm(EditFormValues{Name: "prod"}, nil, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if n.Name != "prod" || n.Host != "" {
		t.Fatalf("unexpected group: %#v", n)
	}
}

func TestApplyEditForm_editGroupNameOnly(t *testing.T) {
	t.Parallel()
	target := &Node{Name: "old", Children: []*Node{{Name: "child", Host: "1.2.3.4"}}}
	n, err := ApplyEditForm(EditFormValues{Name: "new"}, target, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if n.Name != "new" || len(n.Children) != 1 || n.Children[0].Host != "1.2.3.4" {
		t.Fatalf("group rename broke children: %#v", n)
	}
}

func TestCloneNode_deepCopy(t *testing.T) {
	t.Parallel()
	orig := &Node{
		Name: "g",
		Children: []*Node{
			{Name: "a", Host: "1.1.1.1", Port: 22},
			{Name: "b", Host: "2.2.2.2", Jump: []*Node{{Name: "hop", Host: "10.0.0.1"}}},
		},
	}
	cp := CloneNode(orig)
	if cp == orig || cp.Children[0] == orig.Children[0] {
		t.Fatal("expected deep copy")
	}
	if len(cp.Children) != 2 || cp.Children[1].Jump[0] == orig.Children[1].Jump[0] {
		t.Fatalf("clone = %#v", cp)
	}
	cp.Children[0].Name = "changed"
	if orig.Children[0].Name != "a" {
		t.Fatal("mutating clone affected original")
	}
}

func TestUniqueCopyName(t *testing.T) {
	t.Parallel()
	siblings := []*Node{{Name: "web"}, {Name: "web-copy"}}
	if got := UniqueCopyName("web", siblings); got != "web-2" {
		t.Fatalf("got %q, want web-2", got)
	}
	if got := UniqueCopyName("api", nil); got != "api-copy" {
		t.Fatalf("got %q, want api-copy", got)
	}
}

func TestVisibleEditFields(t *testing.T) {
	t.Parallel()
	if len(VisibleEditFields(true)) != 1 {
		t.Fatalf("group fields = %v", VisibleEditFields(true))
	}
	if len(VisibleEditFields(false)) != int(EditFieldCount) {
		t.Fatalf("server fields = %v", VisibleEditFields(false))
	}
}

func loadFixture(t *testing.T, name string) []*Node {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var nodes []*Node
	if err := yaml.Unmarshal(b, &nodes); err != nil {
		t.Fatal(err)
	}
	return nodes
}

func TestFixtures_loadFlat(t *testing.T) {
	t.Parallel()
	nodes := loadFixture(t, "flat.yml")
	if len(nodes) != 2 || !nodes[0].Connectable() {
		t.Fatalf("flat fixture: %#v", nodes)
	}
}

func TestFixtures_loadNestedGroups(t *testing.T) {
	t.Parallel()
	nodes := loadFixture(t, "nested_groups.yml")
	if len(nodes) != 2 || len(nodes[0].Children) != 2 {
		t.Fatalf("nested fixture: %#v", nodes)
	}
}

func TestFixtures_loadMixed(t *testing.T) {
	t.Parallel()
	nodes := loadFixture(t, "mixed.yml")
	if len(nodes) != 3 || nodes[0].Alias != "l" || len(nodes[1].Children) != 1 {
		t.Fatalf("mixed fixture: %#v", nodes)
	}
}
