package tui

import (
	"testing"

	"github.com/yinheli/sshw"
)

func TestEditApplyForm_createLeaf(t *testing.T) {
	t.Parallel()
	e := newEditState()
	e.creating = true
	e.inputs[sshw.EditFieldName].SetValue("web")
	e.inputs[sshw.EditFieldHost].SetValue("10.0.0.1")
	e.inputs[sshw.EditFieldUser].SetValue("deploy")
	e.inputs[sshw.EditFieldPort].SetValue("2222")
	e.inputs[sshw.EditFieldAlias].SetValue("w")

	n, err := e.applyForm()
	if err != nil {
		t.Fatal(err)
	}
	if n.Name != "web" || n.Host != "10.0.0.1" || n.User != "deploy" || n.Port != 2222 || n.Alias != "w" {
		t.Fatalf("unexpected node: %#v", n)
	}
}

func TestEditApplyForm_createGroup(t *testing.T) {
	t.Parallel()
	e := newEditState()
	e.creating = true
	e.inputs[sshw.EditFieldName].SetValue("prod")
	n, err := e.applyForm()
	if err != nil {
		t.Fatal(err)
	}
	if n.Name != "prod" || n.Host != "" || len(n.Children) != 0 {
		t.Fatalf("unexpected group: %#v", n)
	}
}

func TestEditApplyForm_editGroupNameOnly(t *testing.T) {
	t.Parallel()
	target := &sshw.Node{Name: "old", Children: []*sshw.Node{{Name: "child", Host: "1.2.3.4"}}}
	e := newEditState()
	e.target = target
	e.inputs[sshw.EditFieldName].SetValue("new")
	n, err := e.applyForm()
	if err != nil {
		t.Fatal(err)
	}
	if n.Name != "new" || len(n.Children) != 1 {
		t.Fatalf("group rename broke children: %#v", n)
	}
}

func TestRemoveNodeFromCurrentSlice(t *testing.T) {
	t.Parallel()
	a := &sshw.Node{Name: "a", Host: "1.1.1.1"}
	b := &sshw.Node{Name: "b", Host: "2.2.2.2"}
	siblings := []*sshw.Node{a, b}
	for i, n := range siblings {
		if n == a {
			siblings = append(siblings[:i], siblings[i+1:]...)
			break
		}
	}
	if len(siblings) != 1 || siblings[0] != b {
		t.Fatalf("siblings = %#v", siblings)
	}
}
