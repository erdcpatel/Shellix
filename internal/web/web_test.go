package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestWebServerLifecycleAndAPIs(t *testing.T) {
	cfg := ServerConfig{
		Host:      "127.0.0.1",
		Port:      9870,
		Workspace: ".",
	}

	server := NewServer(cfg)
	port, err := server.Start()
	if err != nil {
		t.Fatalf("failed starting server: %v", err)
	}
	defer func() { _ = server.Stop() }()

	baseURL := server.URL()
	if !strings.Contains(baseURL, "127.0.0.1") {
		t.Errorf("unexpected base URL: %s", baseURL)
	}
	if port < 9870 {
		t.Errorf("unexpected port: %d", port)
	}

	// 1. Verify index.html static asset serving
	resp, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("failed requesting root: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for root, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Shellix Workbench") {
		t.Errorf("root missing HTML title: %s", string(body)[:200])
	}

	// 2. Verify /api/info
	infoResp, err := http.Get(baseURL + "/api/info")
	if err != nil {
		t.Fatalf("failed requesting /api/info: %v", err)
	}
	defer infoResp.Body.Close()

	var info map[string]interface{}
	if err := json.NewDecoder(infoResp.Body).Decode(&info); err != nil {
		t.Fatalf("failed decoding /api/info: %v", err)
	}
	if _, ok := info["workspace_name"]; !ok {
		t.Errorf("/api/info missing workspace_name: %+v", info)
	}

	// 3. Verify /api/data/sample for grid caching
	dataResp, err := http.Get(baseURL + "/api/data/sample")
	if err != nil {
		t.Fatalf("failed requesting /api/data/sample: %v", err)
	}
	defer dataResp.Body.Close()

	var sampleData struct {
		Records []map[string]interface{} `json:"records"`
		Cached  bool                     `json:"cached"`
		Count   int                      `json:"count"`
	}
	if err := json.NewDecoder(dataResp.Body).Decode(&sampleData); err != nil {
		t.Fatalf("failed decoding /api/data/sample: %v", err)
	}
	if len(sampleData.Records) == 0 || !sampleData.Cached {
		t.Errorf("unexpected sample records: %+v", sampleData)
	}

	// 4. Verify /api/flows
	flowsResp, err := http.Get(baseURL + "/api/flows")
	if err != nil {
		t.Fatalf("failed requesting /api/flows: %v", err)
	}
	defer flowsResp.Body.Close()
	if flowsResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /api/flows, got %d", flowsResp.StatusCode)
	}

	// 5. Verify /api/search with scope parameter
	searchResp, err := http.Get(baseURL + "/api/search?q=type&scope=internal&ext=go")
	if err != nil {
		t.Fatalf("failed requesting /api/search: %v", err)
	}
	defer searchResp.Body.Close()
	if searchResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /api/search, got %d", searchResp.StatusCode)
	}

	// 6. Verify /api/vault
	vaultResp, err := http.Get(baseURL + "/api/vault?env=default")
	if err != nil {
		t.Fatalf("failed requesting /api/vault: %v", err)
	}
	defer vaultResp.Body.Close()
	if vaultResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /api/vault, got %d", vaultResp.StatusCode)
	}

	// 7. Verify /api/doctor
	doctorResp, err := http.Get(baseURL + "/api/doctor")
	if err != nil {
		t.Fatalf("failed requesting /api/doctor: %v", err)
	}
	defer doctorResp.Body.Close()
	if doctorResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /api/doctor, got %d", doctorResp.StatusCode)
	}

	// 8. Verify /api/http/diff
	httpDiffResp, err := http.Get(baseURL + "/api/http/diff")
	if err != nil {
		t.Fatalf("failed requesting /api/http/diff: %v", err)
	}
	defer httpDiffResp.Body.Close()
	if httpDiffResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /api/http/diff, got %d", httpDiffResp.StatusCode)
	}

	var diffData struct {
		StatusMatch bool           `json:"status_match"`
		AddedKeys   []string       `json:"added_keys"`
		RemovedKeys []string       `json:"removed_keys"`
		SplitRows   []SplitDiffRow `json:"split_rows"`
	}
	if err := json.NewDecoder(httpDiffResp.Body).Decode(&diffData); err != nil {
		t.Fatalf("failed decoding /api/http/diff: %v", err)
	}
	if len(diffData.SplitRows) == 0 {
		t.Errorf("expected split diff rows in /api/http/diff response")
	}

	// 9. Verify /api/data/view (Parquet & Tabular Data Viewer)
	dataViewResp, err := http.Get(baseURL + "/api/data/view")
	if err != nil {
		t.Fatalf("failed requesting /api/data/view: %v", err)
	}
	defer dataViewResp.Body.Close()
	if dataViewResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /api/data/view, got %d", dataViewResp.StatusCode)
	}

	var dataViewSummary struct {
		Format    string                   `json:"format"`
		TotalRows int64                    `json:"total_rows"`
		Columns   []map[string]interface{} `json:"columns"`
		Rows      []map[string]interface{} `json:"rows"`
	}
	if err := json.NewDecoder(dataViewResp.Body).Decode(&dataViewSummary); err != nil {
		t.Fatalf("failed decoding /api/data/view: %v", err)
	}
	if dataViewSummary.Format != "parquet" || dataViewSummary.TotalRows != 20 {
		t.Errorf("unexpected dataview summary: %+v", dataViewSummary)
	}
}

func TestParseUnifiedDiff(t *testing.T) {
	rawDiff := `diff --git a/app.go b/app.go
index 1234..5678 100644
--- a/app.go
+++ b/app.go
@@ -10,4 +10,5 @@
  package main
 
-func oldMethod() {
+func newMethod() {
+	// Added comment
  }
`
	parsed := ParseUnifiedDiff(rawDiff)
	if parsed.Additions != 2 {
		t.Errorf("expected 2 additions, got %d", parsed.Additions)
	}
	if parsed.Deletions != 1 {
		t.Errorf("expected 1 deletion, got %d", parsed.Deletions)
	}
	if len(parsed.SplitRows) == 0 {
		t.Fatalf("expected split rows, got 0")
	}

	foundHunk := false
	for _, r := range parsed.SplitRows {
		if r.IsHunk {
			foundHunk = true
			break
		}
	}
	if !foundHunk {
		t.Errorf("expected hunk row in split rows")
	}
}
