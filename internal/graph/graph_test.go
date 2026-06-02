package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseDir_Empty(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "nodes"), 0755)
	os.MkdirAll(filepath.Join(dir, "edges"), 0755)

	g, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(g.Edges))
	}
}

func TestParseDir_WithNodes(t *testing.T) {
	dir := t.TempDir()
	nodesDir := filepath.Join(dir, "nodes")
	edgesDir := filepath.Join(dir, "edges")
	os.MkdirAll(nodesDir, 0755)
	os.MkdirAll(edgesDir, 0755)

	node := Node{
		ID:             "test_001",
		Type:           NodeTypeConcept,
		Title:          "Test Concept",
		Owner:          "alice",
		Classification: "L1",
		Visibility:     "all",
		Status:         "active",
	}
	data, _ := json.Marshal(node)
	os.WriteFile(filepath.Join(nodesDir, "test_001.json"), data, 0644)

	g, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}
	if g.Nodes[0].Title != "Test Concept" {
		t.Errorf("expected 'Test Concept', got '%s'", g.Nodes[0].Title)
	}
	if g.Nodes[0].Owner != "alice" {
		t.Errorf("expected 'alice', got '%s'", g.Nodes[0].Owner)
	}
}

func TestParseDir_WithEdges(t *testing.T) {
	dir := t.TempDir()
	nodesDir := filepath.Join(dir, "nodes")
	edgesDir := filepath.Join(dir, "edges")
	os.MkdirAll(nodesDir, 0755)
	os.MkdirAll(edgesDir, 0755)

	edges := []Edge{
		{ID: "e1", Source: "a", Target: "b", Relation: "depends_on", Confidence: 0.9},
		{ID: "e2", Source: "b", Target: "c", Relation: "references", Confidence: 0.7},
	}
	data, _ := json.Marshal(edges)
	os.WriteFile(filepath.Join(edgesDir, "relations.json"), data, 0644)

	g, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(g.Edges))
	}
	if g.Edges[0].Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", g.Edges[0].Confidence)
	}
}

func TestParseDir_NonexistentDir(t *testing.T) {
	g, err := ParseDir("/nonexistent/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(g.Nodes))
	}
}

func TestStats(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "n1", Type: NodeTypeConcept, Status: "active", Owner: "alice", Classification: "L1"},
			{ID: "n2", Type: NodeTypeTopic, Status: "draft", Owner: "bob", Classification: "L2"},
			{ID: "n3", Type: NodeTypeConcept, Status: "active", Owner: "alice", Classification: "L1"},
			{ID: "n4", Type: NodeTypeRunbook, Status: "", Owner: "", Classification: ""},
		},
		Edges: []Edge{
			{ID: "e1", Source: "n1", Target: "n2"},
		},
	}

	stats := g.Stats()
	if stats.TotalNodes != 4 {
		t.Errorf("expected 4 nodes, got %d", stats.TotalNodes)
	}
	if stats.TotalEdges != 1 {
		t.Errorf("expected 1 edge, got %d", stats.TotalEdges)
	}
	if stats.TypeBreakdown[NodeTypeConcept] != 2 {
		t.Errorf("expected 2 concepts, got %d", stats.TypeBreakdown[NodeTypeConcept])
	}
	if stats.StatusBreakdown["active"] != 2 {
		t.Errorf("expected 2 active, got %d", stats.StatusBreakdown["active"])
	}
	if stats.StatusBreakdown["unknown"] != 1 {
		t.Errorf("expected 1 unknown, got %d", stats.StatusBreakdown["unknown"])
	}
	if stats.OwnerBreakdown["unset"] != 1 {
		t.Errorf("expected 1 unset owner, got %d", stats.OwnerBreakdown["unset"])
	}
}

func TestParseDir_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	nodesDir := filepath.Join(dir, "nodes")
	os.MkdirAll(nodesDir, 0755)
	os.WriteFile(filepath.Join(nodesDir, "bad.json"), []byte(`{invalid`), 0644)

	g, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes) != 0 {
		t.Errorf("expected 0 nodes for invalid JSON, got %d", len(g.Nodes))
	}
}

func TestParseDir_IDFromFilename(t *testing.T) {
	dir := t.TempDir()
	nodesDir := filepath.Join(dir, "nodes")
	os.MkdirAll(nodesDir, 0755)

	node := Node{Type: NodeTypeFAQ, Title: "FAQ Item"}
	data, _ := json.Marshal(node)
	os.WriteFile(filepath.Join(nodesDir, "faq_042.json"), data, 0644)

	g, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Nodes[0].ID != "faq_042" {
		t.Errorf("expected ID 'faq_042', got '%s'", g.Nodes[0].ID)
	}
}
