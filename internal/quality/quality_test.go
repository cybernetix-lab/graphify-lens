package quality

import (
	"testing"
	"time"

	"github.com/cybernetix-lab/graphify-lens/internal/graph"
)

func makeTestGraph() *graph.Graph {
	now := time.Now().UTC().Format(time.RFC3339)
	oldDate := time.Now().AddDate(0, -3, 0).UTC().Format(time.RFC3339)

	return &graph.Graph{
		Nodes: []graph.Node{
			{
				ID: "n1", Type: graph.NodeTypeConcept, Title: "Concept 1",
				Owner: "alice", Classification: "L1", Visibility: "all",
				Status: "active", LastReviewedAt: now, FreshnessSLA: "720h",
				SourceRefs: []string{"https://example.com/1"},
				ScenarioTags: []string{"oncall"},
			},
			{
				ID: "n2", Type: graph.NodeTypeTopic, Title: "Topic 1",
				Owner: "bob", Classification: "L2", Visibility: "team",
				Status: "active", LastReviewedAt: oldDate, FreshnessSLA: "720h",
				SourceRefs: []string{},
				ScenarioTags: []string{"coding"},
			},
			{
				ID: "n3", Type: graph.NodeTypeRunbook, Title: "Runbook 1",
				Owner: "", Classification: "", Visibility: "",
				Status: "", LastReviewedAt: "",
				SourceRefs: []string{},
				ScenarioTags: []string{},
			},
		},
		Edges: []graph.Edge{
			{ID: "e1", Source: "n1", Target: "n2", Relation: "depends_on", Confidence: 0.9},
			{ID: "e2", Source: "n2", Target: "n3", Relation: "references", Confidence: 0.3},
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
	if a.Coverage.TypeCoverage <= 0 {
		t.Error("type coverage should be > 0 with 3 node types")
	}
}

func TestAssess_Accuracy(t *testing.T) {
	g := makeTestGraph()
	dir := t.TempDir()

	a, err := Assess(g, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Accuracy.GroundedEdgeRate != 0.5 {
		t.Errorf("expected grounded edge rate 0.5, got %f", a.Accuracy.GroundedEdgeRate)
	}
}

func TestAssess_Freshness(t *testing.T) {
	g := makeTestGraph()
	dir := t.TempDir()

	a, err := Assess(g, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Freshness.StalePageRatio <= 0 {
		t.Error("stale page ratio should be > 0 with an old review date")
	}
}

func TestAssess_Governance(t *testing.T) {
	g := makeTestGraph()
	dir := t.TempDir()

	a, err := Assess(g, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Governance.MetadataCompleteness >= 1.0 {
		t.Error("metadata completeness should be < 1.0 with incomplete node")
	}
	if a.Governance.OwnerCoverage < 0.5 {
		t.Error("owner coverage should be >= 0.5")
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
		Timestamp: time.Now(),
		TotalScore: 0.85,
		Version: "1.0",
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
