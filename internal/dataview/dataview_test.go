package dataview

import (
	"bytes"
	"compress/gzip"
	"os"
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
