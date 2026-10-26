package dataview

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InspectData is the primary entry point to automatically decompress and parse any tabular data
func InspectData(rawBytes []byte, filename string, opts InspectOptions) (*DataFileSummary, error) {
	if int64(len(rawBytes)) > MaxDecompressedBytes {
		return nil, fmt.Errorf("data size (%d bytes) exceeds safety limit of %d bytes", len(rawBytes), MaxDecompressedBytes)
	}

	decompressed, err := ReadAndDecompress(rawBytes)
	if err != nil {
		return nil, fmt.Errorf("decompression error: %w", err)
	}

	// Also infer format from filename extension if magic bytes were ambiguous
	format := decompressed.Format
	cleanName := strings.ToLower(filename)
	if strings.HasSuffix(cleanName, ".parquet") || strings.Contains(cleanName, ".parquet.") {
		format = "parquet"
	} else if strings.HasSuffix(cleanName, ".xlsx") {
		format = "xlsx"
	} else if strings.HasSuffix(cleanName, ".tsv") {
		format = "tsv"
	} else if strings.HasSuffix(cleanName, ".csv") {
		format = "csv"
	}

	var summary *DataFileSummary
	switch format {
	case "parquet":
		summary, err = ReadParquetData(decompressed.Data, opts)
	case "xlsx":
		summary, err = ReadExcelData(decompressed.Data, opts)
	case "tsv":
		summary, err = ReadDelimitedData(decompressed.Data, '\t', opts)
	default:
		summary, err = ReadDelimitedData(decompressed.Data, ',', opts)
	}

	if err != nil {
		return nil, err
	}

	summary.FileName = filename
	summary.Compression = decompressed.Compression
	if summary.Format == "" {
		summary.Format = format
	}

	return summary, nil
}

// InspectFile opens a file on disk, automatically handling any compression layers
func InspectFile(filePath string, opts InspectOptions) (*DataFileSummary, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	if fi.Size() > MaxDecompressedBytes {
		return nil, fmt.Errorf("file size (%d bytes) exceeds safety limit of %d bytes", fi.Size(), MaxDecompressedBytes)
	}

	rawBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	base := filepath.Base(filePath)
	return InspectData(rawBytes, base, opts)
}

// ExportRows exports the inspected rows to CSV or JSON
func ExportRows(summary *DataFileSummary, format string) ([]byte, error) {
	if summary == nil || len(summary.Rows) == 0 {
		return []byte(""), nil
	}

	format = strings.ToLower(strings.TrimSpace(format))
	if format == "json" {
		return json.MarshalIndent(summary.Rows, "", "  ")
	}

	// Default CSV export
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	var headers []string
	if len(summary.SelectedColumns) > 0 {
		headers = summary.SelectedColumns
	} else {
		for _, col := range summary.Columns {
			headers = append(headers, col.Name)
		}
	}

	if err := w.Write(headers); err != nil {
		return nil, err
	}

	for _, row := range summary.Rows {
		var record []string
		for _, h := range headers {
			val := row[h]
			record = append(record, fmt.Sprintf("%v", val))
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}
