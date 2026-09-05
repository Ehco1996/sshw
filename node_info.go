package sshw

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
