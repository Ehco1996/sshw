package sshw

import "strings"

// IndexedHost represents a connectable leaf node along with its hierarchy path.
type IndexedHost struct {
	Node       *Node
	Path       []string
	Breadcrumb string
}

// FlattenLeaves returns all connectable leaf nodes in tree order with ancestor paths and breadcrumbs.
func FlattenLeaves(roots []*Node) []IndexedHost {
	var out []IndexedHost
	var walk func([]*Node, []string)
	walk = func(nodes []*Node, ancestors []string) {
		for _, n := range nodes {
			if n.Connectable() {
				out = append(out, IndexedHost{
					Node:       n,
					Path:       ancestors,
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
