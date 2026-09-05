package sshw

import (
	"reflect"
	"testing"
)

func buildTestTree() []*Node {
	return []*Node{
		{
			Name:  "lone",
			Alias: "l",
			Host:  "1.2.3.4",
		},
		{
			Name: "cluster",
			Children: []*Node{
				{
					Name:  "node-a",
					Alias: "na",
					Host:  "10.0.0.1",
				},
				{
					Name: "subcluster",
					Children: []*Node{
						{
							Name:  "node-b",
							Alias: "nb",
							Host:  "10.0.0.2",
						},
					},
				},
			},
		},
		{
			Name:  "backup",
			Alias: "bak",
			Host:  "5.6.7.8",
		},
	}
}

func nodeNames(nodes []*Node) []string {
	var names []string
	for _, n := range nodes {
		names = append(names, n.Name)
	}
	return names
}

func TestMatchTargets(t *testing.T) {
	roots := buildTestTree()

	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{
			name:    "empty pattern",
			pattern: "",
			want:    nil,
		},
		{
			name:    "exact leaf name",
			pattern: "lone",
			want:    []string{"lone"},
		},
		{
			name:    "exact leaf alias",
			pattern: "l",
			want:    []string{"lone"},
		},
		{
			name:    "nested leaf by name",
			pattern: "node-b",
			want:    []string{"node-b"},
		},
		{
			name:    "all keyword",
			pattern: "all",
			want:    []string{"lone", "node-a", "node-b", "backup"},
		},
		{
			name:    "star wildcard",
			pattern: "*",
			want:    []string{"lone", "node-a", "node-b", "backup"},
		},
		{
			name:    "group name expands to children",
			pattern: "cluster",
			want:    []string{"node-a", "node-b"},
		},
		{
			name:    "subgroup name expands to children",
			pattern: "subcluster",
			want:    []string{"node-b"},
		},
		{
			name:    "glob leaf names",
			pattern: "node-*",
			want:    []string{"node-a", "node-b"},
		},
		{
			name:    "comma separated",
			pattern: "lone, backup",
			want:    []string{"lone", "backup"},
		},
		{
			name:    "comma separated with group",
			pattern: "lone, cluster",
			want:    []string{"lone", "node-a", "node-b"},
		},
		{
			name:    "no match",
			pattern: "nonexistent",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchTargets(roots, tt.pattern)
			gotNames := nodeNames(got)
			if !reflect.DeepEqual(gotNames, tt.want) {
				t.Errorf("MatchTargets(%q) = %v, want %v", tt.pattern, gotNames, tt.want)
			}
		})
	}
}
