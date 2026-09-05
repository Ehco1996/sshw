package sshw

import (
	"path/filepath"
	"strings"
)

// GetAllConnectable returns all leaf nodes in tree order that can be connected to.
func GetAllConnectable(roots []*Node) []*Node {
	leaves := FlattenLeaves(roots)
	out := make([]*Node, len(leaves))
	for i, l := range leaves {
		out[i] = l.Node
	}
	return out
}

// CollectConnectableLeaves returns all connectable leaf nodes under a given node (inclusive).
func CollectConnectableLeaves(node *Node) []*Node {
	if node.Connectable() {
		return []*Node{node}
	}
	return GetAllConnectable(node.Children)
}

// MatchTargets resolves a target pattern string into a deduplicated list of connectable leaf nodes.
//
// Pattern can be:
//   - "all" or "*": matches every connectable leaf node.
//   - Comma-separated list of targets (e.g. "dev, prod-*").
//   - Exact name or alias of a leaf node (e.g. "dev", "mz-hk-cp").
//   - Exact name of a group node: expands to all connectable leaf nodes under that group.
//   - Wildcard / glob pattern (e.g. "vultr-*", "mz-*"): matches against Name or Alias.
func MatchTargets(roots []*Node, pattern string) []*Node {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}

	// Comma-separated sub-patterns
	if strings.Contains(pattern, ",") {
		parts := strings.Split(pattern, ",")
		seen := make(map[*Node]struct{})
		var combined []*Node
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			subMatches := matchSingleTarget(roots, part)
			for _, m := range subMatches {
				if _, ok := seen[m]; !ok {
					seen[m] = struct{}{}
					combined = append(combined, m)
				}
			}
		}
		return combined
	}

	return matchSingleTarget(roots, pattern)
}

func matchSingleTarget(roots []*Node, pattern string) []*Node {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}

	all := GetAllConnectable(roots)
	if pattern == "all" || pattern == "*" {
		return all
	}

	// 1. Check exact matches on leaf nodes (Name or Alias) using FindConnectableByNameOrAlias (SSOT)
	exact := FindConnectableByNameOrAlias(roots, pattern)
	if len(exact) > 0 {
		return exact
	}

	var matched []*Node
	seen := make(map[*Node]struct{})
	add := func(n *Node) {
		if _, ok := seen[n]; !ok && n.Connectable() {
			seen[n] = struct{}{}
			matched = append(matched, n)
		}
	}

	// 2. Check exact matches on group nodes
	var walkGroups func([]*Node)
	walkGroups = func(nodes []*Node) {
		for _, n := range nodes {
			if len(n.Children) > 0 {
				if n.Name == pattern {
					leaves := CollectConnectableLeaves(n)
					for _, leaf := range leaves {
						add(leaf)
					}
				}
				walkGroups(n.Children)
			}
		}
	}
	walkGroups(roots)
	if len(matched) > 0 {
		return matched
	}

	// 3. Glob match against leaf Name and Alias, and group Name
	isGlob := strings.ContainsAny(pattern, "*?[]")
	if isGlob {
		pLower := strings.ToLower(pattern)

		// Glob against leaves
		for _, n := range all {
			if matchGlob(pLower, strings.ToLower(n.Name)) ||
				(n.Alias != "" && matchGlob(pLower, strings.ToLower(n.Alias))) {
				add(n)
			}
		}

		// Glob against groups (if group name matches glob, add its leaves)
		var walkGlobGroups func([]*Node)
		walkGlobGroups = func(nodes []*Node) {
			for _, n := range nodes {
				if len(n.Children) > 0 {
					if matchGlob(pLower, strings.ToLower(n.Name)) {
						for _, leaf := range CollectConnectableLeaves(n) {
							add(leaf)
						}
					}
					walkGlobGroups(n.Children)
				}
			}
		}
		walkGlobGroups(roots)
	}

	return matched
}

func matchGlob(pattern, s string) bool {
	matched, err := filepath.Match(pattern, s)
	return err == nil && matched
}
