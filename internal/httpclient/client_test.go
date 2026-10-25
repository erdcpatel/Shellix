package httpclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHttpClientExecution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"count":  42,
		})
	}))
	defer server.Close()

	client := NewClient(5*time.Second, false)
	resp, err := client.Execute(RequestOptions{
		Method:      "GET",
		URL:         server.URL,
		BearerToken: "test-secret-token",
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if !resp.IsJSON {
		t.Errorf("expected response to be recognized as JSON")
	}
	if resp.Timing.TotalTime <= 0 {
		t.Errorf("expected positive total time, got %v", resp.Timing.TotalTime)
	}
}

func TestDiffJSON(t *testing.T) {
	v1 := map[string]interface{}{
		"name":    "Shellix",
		"version": "0.1.0",
		"features": []interface{}{
			"git", "flows", "search",
		},
		"config": map[string]interface{}{
			"theme": "dark",
			"cgo":   false,
		},
	}

	v2 := map[string]interface{}{
		"name":    "Shellix",
		"version": "0.2.0", // changed
		"features": []interface{}{
			"git", "flows", "search", "http", // added
		},
		"config": map[string]interface{}{
			"theme": "dark",
			// cgo removed
			"fast": true, // added
		},
	}

	added, removed, changed := DiffJSON(v1, v2, "$")

	// Verify changed
	foundVersionChange := false
	for _, c := range changed {
		if c.Path == "$.version" && c.Value1 == "0.1.0" && c.Value2 == "0.2.0" {
			foundVersionChange = true
		}
	}
	if !foundVersionChange {
		t.Errorf("expected version change in diff, got %+v", changed)
	}

	// Verify added
	foundFastAdded := false
	foundHttpAdded := false
	for _, a := range added {
		if a == "$.config.fast" {
			foundFastAdded = true
		}
		if a == "$.features[3]" {
			foundHttpAdded = true
		}
	}
	if !foundFastAdded || !foundHttpAdded {
		t.Errorf("expected $.config.fast and $.features[3] added, got %v", added)
	}

	// Verify removed
	foundCgoRemoved := false
	for _, r := range removed {
		if r == "$.config.cgo" {
			foundCgoRemoved = true
		}
	}
	if !foundCgoRemoved {
		t.Errorf("expected $.config.cgo removed, got %v", removed)
	}
}

func TestCompareEndpoints(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"service": "orders",
			"status":  "healthy",
			"version": "v1",
		})
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"service": "orders",
			"status":  "healthy",
			"version": "v2",
			"metrics": "enabled",
		})
	}))
	defer server2.Close()

	client := NewClient(5*time.Second, false)
	res, err := client.CompareEndpoints(server1.URL, server2.URL, RequestOptions{})
	if err != nil {
		t.Fatalf("CompareEndpoints failed: %v", err)
	}

	if !res.StatusMatch {
		t.Errorf("expected status codes to match (both 200)")
	}
	if res.ContentMatch {
		t.Errorf("expected content to differ between v1 and v2")
	}

	report := RenderDiffReport(res)
	if !strings.Contains(report, "metrics") || !strings.Contains(report, "v1") {
		t.Errorf("expected metrics and v1 in report:\n%s", report)
	}
}
