package graph

import (
	"os"
	"path/filepath"
	"testing"
)

func makeTestGraphJSON() []byte {
	return []byte(`{
  "directed": true,
  "multigraph": false,
  "graph": {},
  "nodes": [
    {"id": "n1", "label": "Concept 1", "file_type": "code", "source_file": "src/main.go", "captured_at": "2025-06-01T10:00:00Z", "author": "alice", "contributor": "alice"},
    {"id": "n2", "label": "Document 1", "file_type": "document", "source_file": "docs/readme.md", "captured_at": "2025-03-01T10:00:00Z", "author": "bob", "contributor": "alice"},
    {"id": "n3", "label": "Paper 1", "file_type": "paper", "source_file": "papers/paper.pdf", "source_url": "https://arxiv.org/abs/1234.5678", "captured_at": "2025-01-01T10:00:00Z", "author": "Smith et al."},
    {"id": "n4", "label": "Image 1", "file_type": "image", "source_file": "assets/diagram.png"}
  ],
  "links": [
    {"source": "n1", "target": "n2", "relation": "references", "confidence": "EXTRACTED", "confidence_score": 1.0, "weight": 1.0},
    {"source": "n2", "target": "n3", "relation": "cites", "confidence": "INFERRED", "confidence_score": 0.75, "weight": 0.75},
    {"source": "n3", "target": "n4", "relation": "depicts", "confidence": "AMBIGUOUS", "confidence_score": 0.25, "weight": 0.25}
  ]
}`)
}

func TestParseGraphifyOut_Valid(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "graphify-out")
	os.MkdirAll(outDir, 0755)
	os.WriteFile(filepath.Join(outDir, "graph.json"), makeTestGraphJSON(), 0644)

	g, err := ParseGraphifyOut(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(g.Edges))
	}
	if g.Nodes[0].Label != "Concept 1" {
		t.Errorf("expected 'Concept 1', got '%s'", g.Nodes[0].Label)
	}
	if g.Nodes[0].FileType != "code" {
		t.Errorf("expected 'code', got '%s'", g.Nodes[0].FileType)
	}
	if g.Nodes[0].Author != "alice" {
		t.Errorf("expected 'alice', got '%s'", g.Nodes[0].Author)
	}
}

func TestParseGraphifyOut_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	g, err := ParseGraphifyOut(dir)
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

func TestParseGraphifyOut_NonexistentDir(t *testing.T) {
	g, err := ParseGraphifyOut("/nonexistent/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(g.Nodes))
	}
}

func TestParseGraphifyOut_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "graphify-out")
	os.MkdirAll(outDir, 0755)
	os.WriteFile(filepath.Join(outDir, "graph.json"), []byte(`{invalid`), 0644)

	_, err := ParseGraphifyOut(dir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseGraphifyOut_EmptyGraph(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "graphify-out")
	os.MkdirAll(outDir, 0755)
	os.WriteFile(filepath.Join(outDir, "graph.json"), []byte(`{"directed":true,"multigraph":false,"graph":{},"nodes":[],"links":[]}`), 0644)

	g, err := ParseGraphifyOut(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(g.Nodes))
	}
}

func TestStats(t *testing.T) {
	g := &Graph{
		Nodes: []GraphifyNode{
			{ID: "n1", FileType: "code", Author: "alice"},
			{ID: "n2", FileType: "code", Author: "alice"},
			{ID: "n3", FileType: "document", Author: "bob"},
			{ID: "n4", FileType: "paper", Author: ""},
		},
		Edges: []GraphifyEdge{
			{Source: "n1", Target: "n2", Confidence: "EXTRACTED"},
			{Source: "n2", Target: "n3", Confidence: "INFERRED"},
			{Source: "n3", Target: "n4", Confidence: "AMBIGUOUS"},
		},
	}

	stats := g.Stats()
	if stats.TotalNodes != 4 {
		t.Errorf("expected 4 nodes, got %d", stats.TotalNodes)
	}
	if stats.TotalEdges != 3 {
		t.Errorf("expected 3 edges, got %d", stats.TotalEdges)
	}
	if stats.FileTypeBreakdown["code"] != 2 {
		t.Errorf("expected 2 code nodes, got %d", stats.FileTypeBreakdown["code"])
	}
	if stats.FileTypeBreakdown["document"] != 1 {
		t.Errorf("expected 1 document node, got %d", stats.FileTypeBreakdown["document"])
	}
	if stats.ConfidenceBreakdown["EXTRACTED"] != 1 {
		t.Errorf("expected 1 EXTRACTED edge, got %d", stats.ConfidenceBreakdown["EXTRACTED"])
	}
	if stats.AuthorBreakdown["unset"] != 1 {
		t.Errorf("expected 1 unset author, got %d", stats.AuthorBreakdown["unset"])
	}
}

func TestParseGraphifyOut_ConfidenceScore(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "graphify-out")
	os.MkdirAll(outDir, 0755)
	os.WriteFile(filepath.Join(outDir, "graph.json"), makeTestGraphJSON(), 0644)

	g, err := ParseGraphifyOut(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if g.Edges[0].ConfidenceScore != 1.0 {
		t.Errorf("expected confidence_score 1.0, got %f", g.Edges[0].ConfidenceScore)
	}
	if g.Edges[0].Confidence != "EXTRACTED" {
		t.Errorf("expected EXTRACTED, got %s", g.Edges[0].Confidence)
	}
}
