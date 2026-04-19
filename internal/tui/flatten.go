package tui

import (
	"strings"

	"github.com/yinheli/sshw"
)

// IndexedHost is a connectable leaf with ancestor names from the config root (breadcrumb excludes the leaf name).
type IndexedHost struct {
	Node       *sshw.Node
	Breadcrumb string
}

// FlattenLeaves returns all connectable hosts in tree order with breadcrumb paths.
func FlattenLeaves(roots []*sshw.Node) []IndexedHost {
	var out []IndexedHost
	var walk func([]*sshw.Node, []string)
	walk = func(nodes []*sshw.Node, ancestors []string) {
		for _, n := range nodes {
			if n.Connectable() {
				out = append(out, IndexedHost{
					Node:       n,
					Breadcrumb: strings.Join(ancestors, " ❯ "),
				})
			}
			if len(n.Children) > 0 {
				next := append(append([]string(nil), ancestors...), n.Name)
				walk(n.Children, next)
			}
		}
	}
	walk(roots, nil)
	return out
}
