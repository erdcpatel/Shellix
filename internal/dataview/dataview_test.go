package dataview

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"strings"
	"testing"

	goparquet "github.com/fraugster/parquet-go"
	"github.com/fraugster/parquet-go/parquetschema"
	"github.com/xuri/excelize/v2"
)

func TestDecompressNestedGzipAndParquet(t *testing.T) {
	// 1. Generate a valid in-memory Parquet buffer
	schemaText := `message sample_record {
		required int64 id;
		required binary service (STRING);
		optional double latency_ms;
		optional boolean active;
	}`

	sd, err := parquetschema.ParseSchemaDefinition(schemaText)
	if err != nil {
		t.Fatalf("failed parsing schema: %v", err)
	}

	var parquetBuf bytes.Buffer
	fw := goparquet.NewFileWriter(&parquetBuf, goparquet.WithSchemaDefinition(sd))

	for i := 1; i <= 20; i++ {
		row := map[string]interface{}{
			"id":         int64(i),
			"service":    []byte("auth-service"),
			"latency_ms": float64(i) * 1.5,
			"active":     i%2 == 0,
		}
		if err := fw.AddData(row); err != nil {
			t.Fatalf("failed writing row %d: %v", i, err)
		}
	}
	if err := fw.Close(); err != nil {
		t.Fatalf("failed closing parquet writer: %v", err)
	}

	rawParquet := parquetBuf.Bytes()

	// Save fixture for CLI & Web workbench testing
	_ = os.WriteFile("../../tests/fixtures/sample_metrics.parquet", rawParquet, 0644)

	// 2. Wrap in GZIP layer 1
	var gz1 bytes.Buffer
	w1 := gzip.NewWriter(&gz1)
	_, _ = w1.Write(rawParquet)
	_ = w1.Close()

	// 3. Wrap in GZIP layer 2 (nested gzip: .parquet.gz.gz)
	var gz2 bytes.Buffer
	w2 := gzip.NewWriter(&gz2)
	_, _ = w2.Write(gz1.Bytes())
	_ = w2.Close()

	// 4. Test recursive auto-decompression & inspection
	summary, err := InspectData(gz2.Bytes(), "data.parquet.gz.gz", InspectOptions{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("failed inspecting nested gzip parquet: %v", err)
	}

	if summary.Format != "parquet" {
		t.Errorf("expected format parquet, got %s", summary.Format)
	}
	if summary.Compression != "nested-gzip (2 layers)" {
		t.Errorf("expected 2 layers nested-gzip, got %s", summary.Compression)
	}
	if summary.TotalRows != 20 {
		t.Errorf("expected 20 rows, got %d", summary.TotalRows)
	}
	if len(summary.Rows) != 10 {
		t.Errorf("expected 10 limited rows, got %d", len(summary.Rows))
	}
	if len(summary.Columns) < 4 {
		t.Errorf("expected at least 4 columns, got %d", len(summary.Columns))
	}

	// Verify byte slice conversion to string
	if svc, ok := summary.Rows[0]["service"].(string); !ok || svc != "auth-service" {
		t.Errorf("expected normalized string 'auth-service', got %v (%T)", summary.Rows[0]["service"], summary.Rows[0]["service"])
	}

	// 5. Test Column Projection (Select only 'id' and 'service')
	projSummary, err := InspectData(rawParquet, "data.parquet", InspectOptions{
		Columns: []string{"id", "service"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("failed inspecting projected columns: %v", err)
	}
	if len(projSummary.Rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(projSummary.Rows))
	}
	if _, exists := projSummary.Rows[0]["latency_ms"]; exists {
		t.Errorf("expected latency_ms to be omitted in projected columns")
	}

	// 6. Test Export
	csvBytes, err := ExportRows(projSummary, "csv")
	if err != nil || len(csvBytes) == 0 {
		t.Fatalf("failed exporting to CSV: %v", err)
	}
	jsonBytes, err := ExportRows(projSummary, "json")
	if err != nil || len(jsonBytes) == 0 {
		t.Fatalf("failed exporting to JSON: %v", err)
	}
}

func TestCSVAndTSVInspection(t *testing.T) {
	csvData := `id,name,role,status
101,Alice,Engineer,Active
102,Bob,Architect,Active
103,Charlie,DevOps,Pending`

	summary, err := InspectData([]byte(csvData), "users.csv", InspectOptions{
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("failed inspecting CSV: %v", err)
	}

	if summary.Format != "csv" {
		t.Errorf("expected format csv, got %s", summary.Format)
	}
	if summary.TotalRows != 3 {
		t.Errorf("expected 3 total rows, got %d", summary.TotalRows)
	}
	if len(summary.Rows) != 2 {
		t.Errorf("expected 2 limit rows, got %d", len(summary.Rows))
	}
	if !summary.HasMore {
		t.Errorf("expected has_more to be true")
	}
}

func TestExcelInspection(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "Product")
	_ = f.SetCellValue(sheet, "B1", "Price")
	_ = f.SetCellValue(sheet, "A2", "Gateway")
	_ = f.SetCellValue(sheet, "B2", "199.99")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("failed writing excel: %v", err)
	}

	summary, err := InspectData(buf.Bytes(), "catalog.xlsx", InspectOptions{})
	if err != nil {
		t.Fatalf("failed inspecting excel: %v", err)
	}

	if summary.Format != "xlsx" {
		t.Errorf("expected format xlsx, got %s", summary.Format)
	}
	if summary.TotalRows != 1 {
		t.Errorf("expected 1 data row, got %d", summary.TotalRows)
	}
	if len(summary.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(summary.Columns))
	}
}

func TestDecompressionBombProtection(t *testing.T) {
	// Generate a 1 MB payload of zeroes, compressed with gzip (compresses to ~1 KB)
	largePayload := make([]byte, 1024*1024)
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	_, err := gw.Write(largePayload)
	if err != nil {
		t.Fatalf("failed writing gzip: %v", err)
	}
	_ = gw.Close()

	compressedData := gzBuf.Bytes()

	// 1. With small limit (e.g. 50 KB), decompression should be rejected as a bomb
	_, err = ReadAndDecompressWithLimit(compressedData, 50*1024)
	if err == nil {
		t.Fatalf("expected ErrDecompressionBomb, but decompression succeeded")
	}
	if err != ErrDecompressionBomb {
		t.Errorf("expected ErrDecompressionBomb, got: %v", err)
	}

	// 2. With adequate limit (e.g. 2 MB), decompression should succeed
	res, err := ReadAndDecompressWithLimit(compressedData, 2*1024*1024)
	if err != nil {
		t.Fatalf("expected successful decompression, got: %v", err)
	}
	if len(res.Data) != len(largePayload) {
		t.Errorf("expected %d bytes, got %d", len(largePayload), len(res.Data))
	}
}

func TestZeroRowParquetAndSerialization(t *testing.T) {
	schemaText := `message zero_row_record {
		required int64 id;
		required binary name (STRING);
		optional double score;
	}`

	sd, err := parquetschema.ParseSchemaDefinition(schemaText)
	if err != nil {
		t.Fatalf("failed parsing schema: %v", err)
	}

	var buf bytes.Buffer
	buf.WriteString("PAR1")
	fw := goparquet.NewFileWriter(&buf, goparquet.WithSchemaDefinition(sd))
	// Close without adding any rows
	if err := fw.Close(); err != nil {
		t.Fatalf("failed closing parquet writer: %v", err)
	}

	rawZero := buf.Bytes()
	summary, err := InspectData(rawZero, "zero.parquet", InspectOptions{})
	if err != nil {
		t.Fatalf("failed inspecting 0-row parquet: %v", err)
	}

	if summary.TotalRows != 0 {
		t.Errorf("expected 0 total rows, got %d", summary.TotalRows)
	}
	if summary.Rows == nil {
		t.Errorf("summary.Rows should not be nil")
	}
	if summary.Columns == nil {
		t.Errorf("summary.Columns should not be nil")
	}
	if summary.SelectedColumns == nil {
		t.Errorf("summary.SelectedColumns should not be nil")
	}
	if len(summary.Columns) != 3 {
		t.Errorf("expected 3 schema columns, got %d", len(summary.Columns))
	}

	// Verify JSON serialization produces [] instead of null
	jsonBytes, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("failed marshaling summary: %v", err)
	}
	jsonStr := string(jsonBytes)
	if strings.Contains(jsonStr, `"rows":null`) {
		t.Errorf("JSON should not serialize rows as null: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"rows":[]`) {
		t.Errorf("JSON should serialize rows as []: %s", jsonStr)
	}
	if strings.Contains(jsonStr, `"columns":null`) {
		t.Errorf("JSON should not serialize columns as null: %s", jsonStr)
	}
	if strings.Contains(jsonStr, `"selected_columns":null`) {
		t.Errorf("JSON should not serialize selected_columns as null: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"selected_columns":[]`) {
		t.Errorf("JSON should serialize selected_columns as []: %s", jsonStr)
	}
}

func TestDottedColumnsAndNestedValues(t *testing.T) {
	// 1. Parquet with nested group schema (generates dotted flat names e.g. user.name, metrics.latency_ms)
	schemaText := `message complex_event {
		required int64 event_id;
		required group user {
			required binary name (STRING);
			optional binary email (STRING);
		}
		required group metrics {
			optional double latency_ms;
		}
	}`

	sd, err := parquetschema.ParseSchemaDefinition(schemaText)
	if err != nil {
		t.Fatalf("failed parsing schema: %v", err)
	}

	var buf bytes.Buffer
	fw := goparquet.NewFileWriter(&buf, goparquet.WithSchemaDefinition(sd))

	row := map[string]interface{}{
		"event_id": int64(999),
		"user": map[string]interface{}{
			"name":  []byte("octocat"),
			"email": []byte("octocat@github.com"),
		},
		"metrics": map[string]interface{}{
			"latency_ms": 14.8,
		},
	}
	if err := fw.AddData(row); err != nil {
		t.Fatalf("failed adding row: %v", err)
	}
	if err := fw.Close(); err != nil {
		t.Fatalf("failed closing parquet writer: %v", err)
	}

	summary, err := InspectData(buf.Bytes(), "events.parquet", InspectOptions{})
	if err != nil {
		t.Fatalf("failed inspecting nested parquet: %v", err)
	}

	if summary.TotalRows != 1 {
		t.Errorf("expected 1 row, got %d", summary.TotalRows)
	}

	// Verify dotted column names exist in schema
	foundDotted := false
	for _, c := range summary.Columns {
		if strings.Contains(c.Name, ".") {
			foundDotted = true
			break
		}
	}
	if !foundDotted {
		t.Errorf("expected dotted column flat names, got: %+v", summary.Columns)
	}

	// Verify deep normalization: nested []byte inside map should be string, NOT []byte
	userVal, ok := summary.Rows[0]["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected user to be map[string]interface{}, got: %T (%+v)", summary.Rows[0]["user"], summary.Rows[0]["user"])
	}
	nameVal, ok := userVal["name"].(string)
	if !ok || nameVal != "octocat" {
		t.Errorf("expected normalized nested string 'octocat', got %v (%T)", userVal["name"], userVal["name"])
	}
	emailVal, ok := userVal["email"].(string)
	if !ok || emailVal != "octocat@github.com" {
		t.Errorf("expected normalized nested string 'octocat@github.com', got %v (%T)", userVal["email"], userVal["email"])
	}

	// Verify JSON serialization of nested struct does not contain Base64
	jsonBytes, err := json.Marshal(summary.Rows)
	if err != nil {
		t.Fatalf("failed marshaling rows: %v", err)
	}
	jsonStr := string(jsonBytes)
	if strings.Contains(jsonStr, "b2N0b2NhdA==") { // "octocat" in base64
		t.Errorf("nested value was serialized as Base64 instead of string: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"name":"octocat"`) {
		t.Errorf("expected JSON to contain '\"name\":\"octocat\"', got: %s", jsonStr)
	}

	// Verify CSV export handles dotted columns and nested structures without <nil>
	csvNested, err := ExportRows(summary, "csv")
	if err != nil {
		t.Fatalf("failed exporting nested parquet to CSV: %v", err)
	}
	t.Logf("Nested Parquet CSV Export:\n%s", string(csvNested))
	if strings.Contains(string(csvNested), "<nil>") {
		t.Errorf("CSV export contained <nil> for nested fields: %s", string(csvNested))
	}

	// 2. Test column projection with dotted field
	projSummary, err := InspectData(buf.Bytes(), "events.parquet", InspectOptions{
		Columns: []string{"user.name"},
	})
	if err != nil {
		t.Fatalf("failed projecting user.name: %v", err)
	}
	if len(projSummary.Rows) != 1 {
		t.Fatalf("expected 1 row in projection, got %d", len(projSummary.Rows))
	}
	if projSummary.Rows[0]["user.name"] != "octocat" {
		t.Errorf("expected projected user.name to be 'octocat', got %v", projSummary.Rows[0]["user.name"])
	}

	// 3. Test NormalizeValue edge cases: struct, array, pointer
	type sampleStruct struct {
		Payload []byte `json:"data"`
		ID      int    `json:"id"`
	}
	st := sampleStruct{Payload: []byte("raw-bytes"), ID: 42}
	normStruct, ok := NormalizeValue(st).(map[string]interface{})
	if !ok || normStruct["data"] != "raw-bytes" || normStruct["id"] != 42 {
		t.Errorf("expected struct to be normalized to map with string data, got: %+v", normStruct)
	}

	arr := [4]byte{'t', 'e', 's', 't'}
	if normArr := NormalizeValue(arr); normArr != "test" {
		t.Errorf("expected byte array to normalize to 'test', got: %v", normArr)
	}

	var nilPtr *string
	if normNil := NormalizeValue(nilPtr); normNil != nil {
		t.Errorf("expected nil pointer to normalize to nil, got: %v", normNil)
	}

	// 4. CSV with flat dotted column names
	csvData := `user.id,metrics.latency_ms,system.active
u-101,3.4,true
u-102,5.1,false`

	csvSummary, err := InspectData([]byte(csvData), "telemetry.csv", InspectOptions{})
	if err != nil {
		t.Fatalf("failed inspecting CSV: %v", err)
	}
	if len(csvSummary.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(csvSummary.Columns))
	}
	if csvSummary.Columns[0].Name != "user.id" {
		t.Errorf("expected first column to be 'user.id', got %s", csvSummary.Columns[0].Name)
	}
	if csvSummary.Rows[0]["user.id"] != "u-101" {
		t.Errorf("expected row['user.id'] to be 'u-101', got %v", csvSummary.Rows[0]["user.id"])
	}
}
