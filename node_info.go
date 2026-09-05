package sshw

import "strings"

// NodeInfo is a sanitized representation of a connectable node for public listing and serialization.
// It deliberately omits all credentials (password, passphrase, keypath, etc.).
type NodeInfo struct {
	Name  string   `json:"name"`
	Alias string   `json:"alias,omitempty"`
	Host  string   `json:"host"`
	Port  int      `json:"port"`
	User  string   `json:"user"`
	Jump  string   `json:"jump,omitempty"`
	Path  []string `json:"path,omitempty"`
}

// BuildNodeInfo builds a NodeInfo from a leaf node and its ancestor path.
func BuildNodeInfo(n *Node, path []string) NodeInfo {
	return NodeInfo{
		Name:  n.Name,
		Alias: n.Alias,
		Host:  n.Host,
		Port:  n.SSHPort(),
		User:  n.EffectiveUser(),
		Jump:  n.JumpLabel(),
		Path:  path,
	}
}

// ListNodeInfos uses FlattenLeaves (SSOT) to return NodeInfo for all connectable leaf nodes in tree order.
func ListNodeInfos(roots []*Node) []NodeInfo {
	leaves := FlattenLeaves(roots)
	out := make([]NodeInfo, len(leaves))
	for i, l := range leaves {
		out[i] = BuildNodeInfo(l.Node, l.Path)
	}
	return out
}

// FilterNodeInfos returns NodeInfo for nodes matching targetPattern, retaining their full tree path.
func FilterNodeInfos(roots []*Node, targetPattern string) []NodeInfo {
	targetPattern = strings.TrimSpace(targetPattern)
	if targetPattern == "" {
		return ListNodeInfos(roots)
	}
	matched := MatchTargets(roots, targetPattern)
	if len(matched) == 0 {
		return nil
	}
	matchedSet := make(map[*Node]struct{}, len(matched))
	for _, m := range matched {
		matchedSet[m] = struct{}{}
	}

	leaves := FlattenLeaves(roots)
	var out []NodeInfo
	for _, l := range leaves {
		if _, ok := matchedSet[l.Node]; ok {
			out = append(out, BuildNodeInfo(l.Node, l.Path))
		}
	}
	return out
}
