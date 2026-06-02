package quality

import (
	"testing"
	"time"

	"github.com/cybernetix-lab/graphify-lens/internal/graph"
)

func makeTestGraph() *graph.Graph {
	now := time.Now().UTC().Format(time.RFC3339)
	oldDate := time.Now().AddDate(0, -7, 0).UTC().Format(time.RFC3339)

	return &graph.Graph{
		Nodes: []graph.GraphifyNode{
			{
				ID: "n1", Label: "Concept 1", FileType: "code",
				SourceFile: "src/main.go", CapturedAt: now,
				Author: "alice", Contributor: "alice",
			},
			{
				ID: "n2", Label: "Document 1", FileType: "document",
				SourceFile: "docs/readme.md", CapturedAt: oldDate,
				Author: "bob", Contributor: "alice",
			},
			{
				ID: "n3", Label: "Paper 1", FileType: "paper",
				SourceFile: "papers/paper.pdf", CapturedAt: "",
				Author: "", Contributor: "",
			},
		},
		Edges: []graph.GraphifyEdge{
			{Source: "n1", Target: "n2", Relation: "references", Confidence: "EXTRACTED", ConfidenceScore: 1.0},
			{Source: "n2", Target: "n3", Relation: "cites", Confidence: "INFERRED", ConfidenceScore: 0.75},
			{Source: "n3", Target: "n1", Relation: "conceptually_related_to", Confidence: "AMBIGUOUS", ConfidenceScore: 0.25},
		},
	}
}

func TestAssess_TotalScoreRange(t *testing.T) {
	g := makeTestGraph()
	dir := t.TempDir()

	a, err := Assess(g, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.TotalScore < 0 || a.TotalScore > 1 {
		t.Errorf("total score out of range [0,1]: %f", a.TotalScore)
	}
}

func TestAssess_Coverage(t *testing.T) {
	g := makeTestGraph()
	dir := t.TempDir()

	a, err := Assess(g, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Coverage.Score < 0 || a.Coverage.Score > 1 {
		t.Errorf("coverage score out of range: %f", a.Coverage.Score)
	}
	if a.Coverage.NodeCount != 3 {
		t.Errorf("expected 3 nodes, got %d", a.Coverage.NodeCount)
	}
	if a.Coverage.EdgeCount != 3 {
		t.Errorf("expected 3 edges, got %d", a.Coverage.EdgeCount)
	}
	if a.Coverage.FileTypeCount != 3 {
		t.Errorf("expected 3 file types, got %d", a.Coverage.FileTypeCount)
	}
}

func TestAssess_Accuracy(t *testing.T) {
	g := makeTestGraph()
	dir := t.TempDir()

	a, err := Assess(g, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Accuracy.ExtractedRate != 1.0/3.0 {
		t.Errorf("expected extracted rate 1/3, got %f", a.Accuracy.ExtractedRate)
	}
	if a.Accuracy.InferredRate != 1.0/3.0 {
		t.Errorf("expected inferred rate 1/3, got %f", a.Accuracy.InferredRate)
	}
	if a.Accuracy.AmbiguousRate != 1.0/3.0 {
		t.Errorf("expected ambiguous rate 1/3, got %f", a.Accuracy.AmbiguousRate)
	}
}

func TestAssess_Freshness(t *testing.T) {
	g := makeTestGraph()
	dir := t.TempDir()

	a, err := Assess(g, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Freshness.NodesWithTimestamp != 2 {
		t.Errorf("expected 2 nodes with timestamp, got %d", a.Freshness.NodesWithTimestamp)
	}
	if a.Freshness.StaleNodeRatio <= 0 {
		t.Error("stale node ratio should be > 0 with an old capture date")
	}
}

func TestAssess_Governance(t *testing.T) {
	g := makeTestGraph()
	dir := t.TempDir()

	a, err := Assess(g, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Governance.AuthorCoverage >= 1.0 {
		t.Error("author coverage should be < 1.0 with incomplete node")
	}
	if a.Governance.SourceFileCoverage != 1.0 {
		t.Errorf("expected source_file coverage 1.0, got %f", a.Governance.SourceFileCoverage)
	}
}

func TestAssess_EmptyGraph(t *testing.T) {
	g := &graph.Graph{}
	dir := t.TempDir()

	a, err := Assess(g, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Coverage.Score != 0 {
		t.Errorf("expected coverage 0 for empty graph, got %f", a.Coverage.Score)
	}
	if a.Governance.Score != 1.0 {
		t.Errorf("expected governance 1.0 for empty graph, got %f", a.Governance.Score)
	}
}

func TestSaveAndLoadHistory(t *testing.T) {
	dir := t.TempDir()
	a := &Assessment{
		Timestamp:  time.Now(),
		TotalScore: 0.85,
		Version:    "1.0",
	}

	if err := SaveAssessment(a, dir); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	history, err := LoadHistory(dir)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 assessment, got %d", len(history))
	}
	if history[0].TotalScore != 0.85 {
		t.Errorf("expected score 0.85, got %f", history[0].TotalScore)
	}
}

func TestLoadHistory_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	history, err := LoadHistory(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if history != nil {
		t.Errorf("expected nil for empty dir, got %v", history)
	}
}

func TestLoadHistory_NonexistentDir(t *testing.T) {
	history, err := LoadHistory("/nonexistent/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if history != nil {
		t.Errorf("expected nil for nonexistent dir, got %v", history)
	}
}
