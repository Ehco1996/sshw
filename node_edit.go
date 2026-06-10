package sshw

import (
	"fmt"
	"strconv"
	"strings"
)

// EditField identifies a TUI-editable node attribute. Order matches form layout.
type EditField int

const (
	EditFieldName EditField = iota
	EditFieldHost
	EditFieldUser
	EditFieldPort
	EditFieldAlias
	EditFieldKeyPath
	EditFieldAgentPath
	EditFieldPassphrase
	EditFieldPassword
	EditFieldCount
)

var editFieldLabels = [...]string{
	"name", "host", "user", "port", "alias",
	"keypath", "agentpath", "passphrase", "password",
}

// EditFieldLabel returns the form label for f.
func EditFieldLabel(f EditField) string {
	if f < 0 || int(f) >= len(editFieldLabels) {
		return ""
	}
	return editFieldLabels[f]
}

// EditFieldVisible reports whether f is shown when editing a group vs a server.
func EditFieldVisible(f EditField, isGroup bool) bool {
	if isGroup {
		return f == EditFieldName
	}
	return true
}

// VisibleEditFields returns editable fields for the current form mode.
func VisibleEditFields(isGroup bool) []EditField {
	if isGroup {
		return []EditField{EditFieldName}
	}
	out := make([]EditField, EditFieldCount)
	for i := range out {
		out[i] = EditField(i)
	}
	return out
}

// EditFormValues holds raw string values from the TUI edit form.
type EditFormValues struct {
	Name       string
	Host       string
	User       string
	Port       string
	Alias      string
	KeyPath    string
	AgentPath  string
	Passphrase string
	Password   string
}

// NodeToEditFormValues extracts form values from n.
func NodeToEditFormValues(n *Node) EditFormValues {
	v := EditFormValues{
		Name:       n.Name,
		Host:       n.Host,
		User:       n.User,
		Alias:      n.Alias,
		KeyPath:    n.KeyPath,
		AgentPath:  n.AgentPath,
		Passphrase: n.Passphrase,
		Password:   n.Password,
	}
	if n.Port > 0 {
		v.Port = strconv.Itoa(n.Port)
	}
	return v
}

// IsGroupForm reports whether the edit form represents a group (directory) node.
func IsGroupForm(target *Node, hostInput string, creating bool) bool {
	if target != nil && len(target.Children) > 0 {
		return true
	}
	if creating && strings.TrimSpace(hostInput) == "" {
		return true
	}
	return false
}

// ApplyEditForm validates values and applies them to a node.
// When creating, a new node is returned; otherwise target is updated in place.
func ApplyEditForm(values EditFormValues, target *Node, creating, isGroup bool) (*Node, error) {
	name := strings.TrimSpace(values.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	host := strings.TrimSpace(values.Host)
	user := strings.TrimSpace(values.User)
	alias := strings.TrimSpace(values.Alias)
	portStr := strings.TrimSpace(values.Port)

	var port int
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil || p < 0 {
			return nil, fmt.Errorf("port must be a non-negative number")
		}
		port = p
	}

	if !isGroup && host == "" {
		return nil, fmt.Errorf("host is required for a server entry")
	}

	var n *Node
	if creating {
		n = &Node{Name: name}
	} else {
		n = target
		n.Name = name
	}

	if isGroup {
		if !creating {
			return n, nil
		}
		n.Host = ""
		n.User = ""
		n.Port = 0
		n.Alias = ""
		n.KeyPath = ""
		n.AgentPath = ""
		n.Passphrase = ""
		n.Password = ""
		return n, nil
	}

	n.Host = host
	n.User = user
	n.Port = port
	n.Alias = alias
	n.KeyPath = strings.TrimSpace(values.KeyPath)
	n.AgentPath = strings.TrimSpace(values.AgentPath)
	n.Passphrase = values.Passphrase
	n.Password = values.Password
	return n, nil
}

// CloneNode returns a deep copy of n, including Children and Jump subtrees.
func CloneNode(n *Node) *Node {
	if n == nil {
		return nil
	}
	cp := *n
	if len(n.Children) > 0 {
		cp.Children = make([]*Node, len(n.Children))
		for i, c := range n.Children {
			cp.Children[i] = CloneNode(c)
		}
	}
	if len(n.Jump) > 0 {
		cp.Jump = make([]*Node, len(n.Jump))
		for i, j := range n.Jump {
			cp.Jump[i] = CloneNode(j)
		}
	}
	if len(n.CallbackShells) > 0 {
		cp.CallbackShells = make([]*CallbackShell, len(n.CallbackShells))
		for i, cs := range n.CallbackShells {
			c := *cs
			cp.CallbackShells[i] = &c
		}
	}
	return &cp
}

// UniqueCopyName picks a sibling name for a duplicated node: base-copy, then base-2, …
func UniqueCopyName(base string, siblings []*Node) string {
	used := make(map[string]struct{}, len(siblings))
	for _, s := range siblings {
		used[s.Name] = struct{}{}
	}
	for _, candidate := range []string{base + "-copy", base + "-2", base + "-3", base + "-4", base + "-5"} {
		if _, ok := used[candidate]; !ok {
			return candidate
		}
	}
	for i := 6; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := used[candidate]; !ok {
			return candidate
		}
	}
	return base + "-copy"
}
