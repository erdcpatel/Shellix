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

	var selectedCols []string
	if len(opts.Columns) > 0 {
		selectedCols = opts.Columns
	}

	fr, err := goparquet.NewFileReader(reader, selectedCols...)
	if err != nil {
		return nil, fmt.Errorf("failed opening parquet reader: %w", err)
	}

	totalRows := fr.NumRows()

	var allCols []ColumnMeta
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

	var rows []map[string]interface{}
	for len(rows) < limit {
		row, err := fr.NextRow()
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}

		// Normalize byte slices to strings for human readability
		normalized := make(map[string]interface{})
		for k, v := range row {
			if b, ok := v.([]byte); ok {
				normalized[k] = string(b)
			} else {
				normalized[k] = v
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
