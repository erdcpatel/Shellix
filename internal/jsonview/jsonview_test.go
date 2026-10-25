package jsonview

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateJSONPath(t *testing.T) {
	rawJSON := `{
		"app": "shellix",
		"version": "1.0.0",
		"config": {
			"port": 8080,
			"enabled": true
		},
		"tags": ["cli", "terminal", "devops"],
		"users": [
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"}
		]
	}`

	r := strings.NewReader(rawJSON)
	data, err := QueryFromReader(r, ".")
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	tests := []struct {
		path    string
		wantStr string
		wantErr bool
	}{
		{".app", "shellix", false},
		{".config.port", "8080", false},
		{".tags[1]", "terminal", false},
		{".users[0].name", "Alice", false},
		{".users[*].id", "[1,2]", false},
		{".missing.key", "", true},
		{".tags[10]", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			res, err := EvaluateJSONPath(data, tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("EvaluateJSONPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
			if !tt.wantErr {
				formatted, _ := FormatQueryResult(res, false)
				if formatted != tt.wantStr {
					t.Errorf("got %q, want %q", formatted, tt.wantStr)
				}
			}
		})
	}
}

func TestColorizeJSON(t *testing.T) {
	raw := []byte(`{"status":"ok","code":200,"active":true,"data":null}`)
	out, err := ColorizeJSON(raw)
	if err != nil {
		t.Fatalf("ColorizeJSON failed: %v", err)
	}

	if !strings.Contains(out, "status") || !strings.Contains(out, "200") {
		t.Errorf("unexpected colorized output: %s", out)
	}
}

func TestCompareJSONBytes(t *testing.T) {
	doc1 := []byte(`{
		"name": "Shellix",
		"version": "1.0.0",
		"deprecated": "old_field",
		"features": ["http", "vault"]
	}`)

	doc2 := []byte(`{
		"name": "Shellix",
		"version": "1.1.0",
		"new_feature": "json_viewer",
		"features": ["http", "vault", "cron"]
	}`)

	res, err := CompareJSONBytes(doc1, doc2, "doc1.json", "doc2.json")
	if err != nil {
		t.Fatalf("CompareJSONBytes failed: %v", err)
	}

	if res.Identical {
		t.Errorf("expected differences between documents")
	}

	// Verify added keys: new_feature, features[2]
	if len(res.AddedKeys) != 2 {
		t.Errorf("expected 2 added keys, got %d: %v", len(res.AddedKeys), res.AddedKeys)
	}

	// Verify removed keys: deprecated
	if len(res.RemovedKeys) != 1 || res.RemovedKeys[0] != "deprecated" {
		t.Errorf("expected removed key 'deprecated', got %v", res.RemovedKeys)
	}

	// Verify changed values: version
	if len(res.ChangedValues) != 1 || res.ChangedValues[0].Path != "version" {
		t.Errorf("expected changed value at 'version', got %v", res.ChangedValues)
	}

	report := res.RenderDiffReport()
	if !strings.Contains(report, "JSON Structural Diff Report") {
		t.Errorf("invalid diff report output")
	}
}

func TestStreamLargeFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-fileview-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.log")
	content := "Line 1: init\nLine 2: loading config\nLine 3: server listening\nLine 4: client connected\nLine 5: done\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Test forward stream with max 3 lines
	var buf bytes.Buffer
	cfg := ViewerConfig{
		FilePath:        filePath,
		MaxLines:        3,
		ShowLineNumbers: true,
	}
	err = StreamLargeFile(context.Background(), &buf, cfg)
	if err != nil {
		t.Fatalf("StreamLargeFile failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Line 1") || !strings.Contains(output, "Line 3") || strings.Contains(output, "Line 4") {
		t.Errorf("unexpected max lines output: %s", output)
	}

	// Test tail seek last 2 lines
	var tailBuf bytes.Buffer
	tailCfg := ViewerConfig{
		FilePath:        filePath,
		MaxLines:        2,
		Tail:            true,
		SearchPattern:   "done",
		ShowLineNumbers: true,
	}
	err = StreamLargeFile(context.Background(), &tailBuf, tailCfg)
	if err != nil {
		t.Fatalf("StreamLargeFile tail failed: %v", err)
	}

	tailOut := tailBuf.String()
	if !strings.Contains(tailOut, "Line 4") || !strings.Contains(tailOut, "Line 5") || strings.Contains(tailOut, "Line 1") {
		t.Errorf("unexpected tail output: %s", tailOut)
	}
}

func TestJSONTUIModel(t *testing.T) {
	raw := []byte(`{
		"server": "api-cluster",
		"port": 9000,
		"services": [
			{"name": "auth", "active": true},
			{"name": "billing", "active": false}
		]
	}`)

	m, err := NewJSONTUIModel(raw, "test.json")
	if err != nil {
		t.Fatalf("failed creating JSON TUI model: %v", err)
	}

	if len(m.visibleRows) == 0 {
		t.Fatalf("expected visible rows, got 0")
	}

	view := m.View()
	if !strings.Contains(view, "Shellix Interactive JSON Explorer") || !strings.Contains(view, "server") {
		t.Errorf("unexpected initial view: %s", view)
	}

	// Test expand all
	setAllExpanded(m.root, true)
	m.rebuildVisibleRows()
	expandedCount := len(m.visibleRows)

	// Test collapse all
	setAllExpanded(m.root, false)
	m.root.Expanded = true
	m.rebuildVisibleRows()
	collapsedCount := len(m.visibleRows)

	if collapsedCount >= expandedCount {
		t.Errorf("expected collapsed count (%d) < expanded count (%d)", collapsedCount, expandedCount)
	}
}

