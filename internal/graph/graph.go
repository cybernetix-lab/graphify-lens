package graph

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type GraphifyNode struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	FileType       string `json:"file_type"`
	SourceFile     string `json:"source_file"`
	SourceLocation string `json:"source_location,omitempty"`
	SourceURL      string `json:"source_url,omitempty"`
	CapturedAt     string `json:"captured_at,omitempty"`
	Author         string `json:"author,omitempty"`
	Contributor    string `json:"contributor,omitempty"`
}

type GraphifyEdge struct {
	Source          string  `json:"source"`
	Target          string  `json:"target"`
	Relation        string  `json:"relation"`
	Confidence      string  `json:"confidence"`
	ConfidenceScore float64 `json:"confidence_score"`
	SourceFile      string  `json:"source_file,omitempty"`
	SourceLocation  string  `json:"source_location,omitempty"`
	Weight          float64 `json:"weight"`
}

type GraphifyData struct {
	Directed   bool           `json:"directed"`
	Multigraph bool           `json:"multigraph"`
	Graph      map[string]any `json:"graph"`
	Nodes      []GraphifyNode `json:"nodes"`
	Links      []GraphifyEdge `json:"links"`
}

type Graph struct {
	Nodes     []GraphifyNode `json:"nodes"`
	Edges     []GraphifyEdge `json:"edges"`
	UpdatedAt time.Time      `json:"updated_at"`
	Version   string         `json:"version"`
}

type GraphSummary struct {
	TotalNodes int            `json:"total_nodes"`
	TotalEdges int            `json:"total_edges"`
	FileTypes  map[string]int `json:"file_types"`
}

type GraphQuery struct {
	Query    string `json:"query"`
	FileType string `json:"file_type"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

type GraphResult struct {
	Nodes      []GraphifyNode `json:"nodes"`
	Edges      []GraphifyEdge `json:"edges"`
	TotalNodes int            `json:"total_nodes"`
	TotalEdges int            `json:"total_edges"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
	Truncated  bool           `json:"truncated"`
}

const (
	DefaultLimit = 500
	MaxLimit     = 5000
)

func ParseGraphifyOut(workDir string) (*Graph, error) {
	graphPath := filepath.Join(workDir, "graphify-out", "graph.json")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Graph{
				Nodes:     make([]GraphifyNode, 0),
				Edges:     make([]GraphifyEdge, 0),
				UpdatedAt: time.Now(),
				Version:   "1.0",
			}, nil
		}
		return nil, err
	}

	var raw GraphifyData
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	g := &Graph{
		Nodes:     raw.Nodes,
		Edges:     raw.Links,
		UpdatedAt: time.Now(),
		Version:   "1.0",
	}

	if g.Nodes == nil {
		g.Nodes = make([]GraphifyNode, 0)
	}
	if g.Edges == nil {
		g.Edges = make([]GraphifyEdge, 0)
	}

	return g, nil
}

func ParseGraphifyOutSummary(workDir string) (*GraphSummary, error) {
	graphPath := filepath.Join(workDir, "graphify-out", "graph.json")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &GraphSummary{}, nil
		}
		return nil, err
	}

	var raw GraphifyData
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	summary := &GraphSummary{
		TotalNodes: len(raw.Nodes),
		TotalEdges: len(raw.Links),
		FileTypes:  make(map[string]int),
	}
	for _, n := range raw.Nodes {
		ft := n.FileType
		if ft == "" {
			ft = "unknown"
		}
		summary.FileTypes[ft]++
	}
	return summary, nil
}

func ParseGraphifyOutQuery(workDir string, q GraphQuery) (*GraphResult, error) {
	graphPath := filepath.Join(workDir, "graphify-out", "graph.json")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &GraphResult{
				Nodes: make([]GraphifyNode, 0),
				Edges: make([]GraphifyEdge, 0),
			}, nil
		}
		return nil, err
	}

	var raw GraphifyData
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	if q.Limit <= 0 || q.Limit > MaxLimit {
		q.Limit = DefaultLimit
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	queryLower := strings.ToLower(q.Query)
	fileTypeLower := strings.ToLower(q.FileType)

	var matchedNodes []GraphifyNode
	matchedIDs := make(map[string]bool)

	for _, n := range raw.Nodes {
		if fileTypeLower != "" && strings.ToLower(n.FileType) != fileTypeLower {
			continue
		}
		if queryLower != "" {
			if !strings.Contains(strings.ToLower(n.Label), queryLower) &&
				!strings.Contains(strings.ToLower(n.ID), queryLower) &&
				!strings.Contains(strings.ToLower(n.SourceFile), queryLower) {
				continue
			}
		}
		matchedNodes = append(matchedNodes, n)
		matchedIDs[n.ID] = true
	}

	totalMatched := len(matchedNodes)

	end := q.Offset + q.Limit
	if end > len(matchedNodes) {
		end = len(matchedNodes)
	}
	if q.Offset > len(matchedNodes) {
		q.Offset = len(matchedNodes)
		end = len(matchedNodes)
	}
	pagedNodes := matchedNodes[q.Offset:end]

	var pagedEdges []GraphifyEdge
	for _, e := range raw.Links {
		if matchedIDs[e.Source] && matchedIDs[e.Target] {
			pagedEdges = append(pagedEdges, e)
		}
	}

	truncated := q.Offset+q.Limit < totalMatched

	return &GraphResult{
		Nodes:      pagedNodes,
		Edges:      pagedEdges,
		TotalNodes: totalMatched,
		TotalEdges: len(pagedEdges),
		Limit:      q.Limit,
		Offset:     q.Offset,
		Truncated:  truncated,
	}, nil
}

func (g *Graph) Stats() GraphStats {
	stats := GraphStats{
		TotalNodes:          len(g.Nodes),
		TotalEdges:          len(g.Edges),
		FileTypeBreakdown:   make(map[string]int),
		ConfidenceBreakdown: make(map[string]int),
		AuthorBreakdown:     make(map[string]int),
	}

	for _, n := range g.Nodes {
		ft := n.FileType
		if ft == "" {
			ft = "unknown"
		}
		stats.FileTypeBreakdown[ft]++

		author := n.Author
		if author == "" {
			author = "unset"
		}
		stats.AuthorBreakdown[author]++
	}

	for _, e := range g.Edges {
		conf := e.Confidence
		if conf == "" {
			conf = "UNKNOWN"
		}
		stats.ConfidenceBreakdown[conf]++
	}

	return stats
}

type GraphStats struct {
	TotalNodes          int            `json:"total_nodes"`
	TotalEdges          int            `json:"total_edges"`
	FileTypeBreakdown   map[string]int `json:"file_type_breakdown"`
	ConfidenceBreakdown map[string]int `json:"confidence_breakdown"`
	AuthorBreakdown     map[string]int `json:"author_breakdown"`
}

type Snapshot struct {
	Version     string          `json:"version"`
	Timestamp   time.Time       `json:"timestamp"`
	Checksum    string          `json:"checksum"`
	Summary     GraphSummary    `json:"summary"`
	Topology    TopologyFingerprint `json:"topology"`
	TopNodes    []TopNode       `json:"top_nodes"`
	EdgeStats   EdgeStats       `json:"edge_stats"`
	QualityRef  string          `json:"quality_ref,omitempty"`
}

type TopologyFingerprint struct {
	AvgDegree        float64 `json:"avg_degree"`
	MaxDegree        int     `json:"max_degree"`
	ConnectedComponents int  `json:"connected_components"`
	IsolatedNodes    int     `json:"isolated_nodes"`
}

type TopNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	FileType string `json:"file_type"`
	Degree   int    `json:"degree"`
}

type EdgeStats struct {
	RelationBreakdown map[string]int `json:"relation_breakdown"`
	AvgConfidence     float64        `json:"avg_confidence"`
}

func GenerateSnapshot(workDir string) (*Snapshot, error) {
	graphPath := filepath.Join(workDir, "graphify-out", "graph.json")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		return nil, err
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256(data))

	var raw GraphifyData
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	nodeIndex := make(map[string]*GraphifyNode, len(raw.Nodes))
	for i := range raw.Nodes {
		nodeIndex[raw.Nodes[i].ID] = &raw.Nodes[i]
	}

	degreeMap := make(map[string]int)
	for _, e := range raw.Links {
		degreeMap[e.Source]++
		degreeMap[e.Target]++
	}

	var maxDegree int
	connectedCount := 0
	for _, deg := range degreeMap {
		if deg > 0 {
			connectedCount++
		}
		if deg > maxDegree {
			maxDegree = deg
		}
	}
	isolated := len(raw.Nodes) - connectedCount

	var avgDegree float64
	if len(raw.Nodes) > 0 {
		avgDegree = float64(len(raw.Links)*2) / float64(len(raw.Nodes))
	}

	type nodeDegree struct {
		node   GraphifyNode
		degree int
	}
	ranked := make([]nodeDegree, 0, len(raw.Nodes))
	for _, n := range raw.Nodes {
		ranked = append(ranked, nodeDegree{n, degreeMap[n.ID]})
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].degree > ranked[j].degree
	})

	topN := 20
	if len(ranked) < topN {
		topN = len(ranked)
	}
	topNodes := make([]TopNode, topN)
	for i := 0; i < topN; i++ {
		topNodes[i] = TopNode{
			ID:       ranked[i].node.ID,
			Label:    ranked[i].node.Label,
			FileType: ranked[i].node.FileType,
			Degree:   ranked[i].degree,
		}
	}

	relationBreakdown := make(map[string]int)
	var totalConf float64
	for _, e := range raw.Links {
		relationBreakdown[e.Relation]++
		totalConf += e.ConfidenceScore
	}
	var avgConf float64
	if len(raw.Links) > 0 {
		avgConf = totalConf / float64(len(raw.Links))
	}

	fileTypes := make(map[string]int)
	for _, n := range raw.Nodes {
		ft := n.FileType
		if ft == "" {
			ft = "unknown"
		}
		fileTypes[ft]++
	}

	cc := estimateConnectedComponents(len(raw.Nodes), len(raw.Links), isolated)

	return &Snapshot{
		Version:   "1.0",
		Timestamp: time.Now(),
		Checksum:  checksum,
		Summary: GraphSummary{
			TotalNodes: len(raw.Nodes),
			TotalEdges: len(raw.Links),
			FileTypes:  fileTypes,
		},
		Topology: TopologyFingerprint{
			AvgDegree:           avgDegree,
			MaxDegree:           maxDegree,
			ConnectedComponents: cc,
			IsolatedNodes:       isolated,
		},
		TopNodes: topNodes,
		EdgeStats: EdgeStats{
			RelationBreakdown: relationBreakdown,
			AvgConfidence:     avgConf,
		},
	}, nil
}

func estimateConnectedComponents(totalNodes, totalEdges, isolated int) int {
	if totalNodes == 0 {
		return 0
	}
	connected := totalNodes - isolated
	if connected <= 1 {
		return isolated + connected
	}
	excess := totalEdges - (connected - 1)
	if excess <= 0 {
		return isolated + 1
	}
	estimated := isolated + 1
	if excess > 0 {
		estimated = isolated + 1
	}
	_ = excess
	return estimated
}

func SaveSnapshot(s *Snapshot, workDir string) error {
	snapshotPath := filepath.Join(workDir, "graphify-out", "snapshot.json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(snapshotPath, data, 0644)
}

func LoadSnapshot(workDir string) (*Snapshot, error) {
	snapshotPath := filepath.Join(workDir, "graphify-out", "snapshot.json")
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return nil, err
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
