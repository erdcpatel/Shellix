package dataview

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	goparquet "github.com/fraugster/parquet-go"
)

// ReadParquetData parses a Parquet byte slice and returns structured metadata, schema, and rows
func ReadParquetData(data []byte, opts InspectOptions) (*DataFileSummary, error) {
	if int64(len(data)) > MaxDecompressedBytes {
		return nil, fmt.Errorf("parquet data exceeds maximum allowed size (%d bytes)", MaxDecompressedBytes)
	}

	reader := bytes.NewReader(data)

	selectedCols := make([]string, 0)
	if len(opts.Columns) > 0 {
		selectedCols = opts.Columns
	}

	fr, err := goparquet.NewFileReader(reader, selectedCols...)
	if err != nil {
		return nil, fmt.Errorf("failed opening parquet reader: %w", err)
	}

	totalRows := fr.NumRows()

	allCols := make([]ColumnMeta, 0)
	colNames := make(map[string]bool)

	// Extract schema columns
	for _, col := range fr.Columns() {
		name := col.FlatName()
		if name == "" {
			name = col.Name()
		}
		if name == "" || colNames[name] {
			continue
		}
		colNames[name] = true

		typeStr := "UNKNOWN"
		if col.Type() != nil {
			typeStr = col.Type().String()
		}

		allCols = append(allCols, ColumnMeta{
			Name:     name,
			Type:     typeStr,
			Nullable: true,
		})
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	// Skip up to offset
	for i := 0; i < offset; i++ {
		_, err := fr.NextRow()
		if err != nil {
			break
		}
	}

	rows := make([]map[string]interface{}, 0)
	for len(rows) < limit {
		row, err := fr.NextRow()
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}

		// Normalize byte slices to strings and deeply normalize nested structures
		normalized := make(map[string]interface{}, len(row))
		for k, v := range row {
			normalized[k] = NormalizeValue(v)
		}

		// Also populate flattened dotted keys for nested structs so field lookups
		// (e.g. "user.name", "metrics.latency_ms") succeed directly in Tabulator and ExportRows
		for k, v := range normalized {
			if m, ok := v.(map[string]interface{}); ok {
				flattenNestedMap(k, m, normalized)
			}
		}

		// Apply optional row-level text filter
		if opts.Filter != "" {
			match := false
			fLower := strings.ToLower(opts.Filter)
			for _, val := range normalized {
				if strings.Contains(strings.ToLower(fmt.Sprintf("%v", val)), fLower) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		rows = append(rows, normalized)
	}

	hasMore := int64(offset+len(rows)) < totalRows

	return &DataFileSummary{
		Format:          "parquet",
		TotalRows:       totalRows,
		TotalCols:       len(allCols),
		FileSize:        int64(len(data)),
		Columns:         allCols,
		SelectedColumns: selectedCols,
		Rows:            rows,
		Offset:          offset,
		Limit:           limit,
		HasMore:         hasMore,
	}, nil
}

func flattenNestedMap(prefix string, m map[string]interface{}, target map[string]interface{}) {
	for k, v := range m {
		fullKey := prefix + "." + k
		if subMap, ok := v.(map[string]interface{}); ok {
			flattenNestedMap(fullKey, subMap, target)
		} else {
			if _, exists := target[fullKey]; !exists {
				target[fullKey] = v
			}
		}
	}
}

