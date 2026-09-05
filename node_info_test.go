package sshw

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestListNodeInfos(t *testing.T) {
	roots := []*Node{
		{
			Name:       "node1",
			Alias:      "n1",
			Host:       "1.1.1.1",
			User:       "admin",
			Port:       2222,
			Password:   "secret123",
			Passphrase: "keysupersecret",
		},
		{
			Name: "mygroup",
			Children: []*Node{
				{
					Name: "node2",
					Host: "2.2.2.2",
					// User and Port empty to test defaults
				},
			},
		},
	}

	infos := ListNodeInfos(roots)
	if len(infos) != 2 {
		t.Fatalf("expected 2 infos, got %d", len(infos))
	}

	// Check node1
	if infos[0].Name != "node1" || infos[0].Alias != "n1" || infos[0].Host != "1.1.1.1" ||
		infos[0].Port != 2222 || infos[0].User != "admin" || len(infos[0].Path) != 0 {
		t.Errorf("unexpected info[0]: %+v", infos[0])
	}

	// Check node2 defaults & path
	if infos[1].Name != "node2" || infos[1].Host != "2.2.2.2" ||
		infos[1].Port != 22 || infos[1].User != "root" || len(infos[1].Path) != 1 || infos[1].Path[0] != "mygroup" {
		t.Errorf("unexpected info[1]: %+v", infos[1])
	}

	// Check JSON serialization omits secrets
	data, err := json.Marshal(infos)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	raw := string(data)
	if strings.Contains(raw, "secret123") || strings.Contains(raw, "keysupersecret") {
		t.Fatalf("JSON leaked credentials: %s", raw)
	}
}

func TestFilterNodeInfos(t *testing.T) {
	roots := []*Node{
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
					Name:  "node-b",
					Alias: "nb",
					Host:  "10.0.0.2",
				},
			},
		},
	}

	// 1. Empty pattern returns all
	all := FilterNodeInfos(roots, "")
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}

	// 2. Filter by group name preserves Path
	groupInfos := FilterNodeInfos(roots, "cluster")
	if len(groupInfos) != 2 {
		t.Fatalf("expected 2 in cluster, got %d", len(groupInfos))
	}
	if groupInfos[0].Name != "node-a" || len(groupInfos[0].Path) != 1 || groupInfos[0].Path[0] != "cluster" {
		t.Errorf("expected node-a under cluster, got %+v", groupInfos[0])
	}
	if groupInfos[1].Name != "node-b" || len(groupInfos[1].Path) != 1 || groupInfos[1].Path[0] != "cluster" {
		t.Errorf("expected node-b under cluster, got %+v", groupInfos[1])
	}

	// 3. Filter by alias
	aliasInfos := FilterNodeInfos(roots, "l")
	if len(aliasInfos) != 1 || aliasInfos[0].Name != "lone" {
		t.Errorf("expected lone, got %+v", aliasInfos)
	}

	// 4. Non-matching pattern returns empty
	emptyInfos := FilterNodeInfos(roots, "nonexistent")
	if len(emptyInfos) != 0 {
		t.Errorf("expected empty, got %+v", emptyInfos)
	}
}
