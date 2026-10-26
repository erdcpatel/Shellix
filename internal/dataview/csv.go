package dataview

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// ReadDelimitedData parses CSV or TSV data with column filtering and pagination
func ReadDelimitedData(data []byte, delimiter rune, opts InspectOptions) (*DataFileSummary, error) {
	if int64(len(data)) > MaxDecompressedBytes {
		return nil, fmt.Errorf("delimited data exceeds maximum allowed size (%d bytes)", MaxDecompressedBytes)
	}

	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = delimiter
	r.LazyQuotes = true
	format := "csv"
	if delimiter == '\t' {
		format = "tsv"
	}

	allCols := make([]ColumnMeta, 0)
	activeCols := make([]string, 0)
	rows := make([]map[string]interface{}, 0)

	headers, err := r.Read()
	if err != nil {
		if err == io.EOF {
			return &DataFileSummary{
				Format:          format,
				TotalRows:       0,
				TotalCols:       0,
				FileSize:        int64(len(data)),
				Columns:         allCols,
				SelectedColumns: activeCols,
				Rows:            rows,
				Offset:          opts.Offset,
				Limit:           opts.Limit,
				HasMore:         false,
			}, nil
		}
		return nil, fmt.Errorf("failed reading CSV headers: %w", err)
	}

	for _, h := range headers {
		cleanHeader := strings.TrimSpace(h)
		if cleanHeader == "" {
			cleanHeader = fmt.Sprintf("col_%d", len(allCols)+1)
		}
		allCols = append(allCols, ColumnMeta{
			Name:     cleanHeader,
			Type:     "STRING",
			Nullable: true,
		})
	}

	colIdxMap := make(map[string]int)
	for i, c := range allCols {
		colIdxMap[c.Name] = i
	}

	if len(opts.Columns) > 0 {
		activeCols = opts.Columns
	} else {
		for _, c := range allCols {
			activeCols = append(activeCols, c.Name)
		}
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	totalRows := int64(0)
	fLower := strings.ToLower(opts.Filter)

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		totalRows++

		if totalRows <= int64(offset) {
			continue
		}

		if len(rows) >= limit {
			continue
		}

		rowMap := make(map[string]interface{})
		match := opts.Filter == ""

		for _, colName := range activeCols {
			idx, exists := colIdxMap[colName]
			val := ""
			if exists && idx < len(record) {
				val = record[idx]
			}
			rowMap[colName] = val

			if !match && strings.Contains(strings.ToLower(val), fLower) {
				match = true
			}
		}

		if match {
			rows = append(rows, rowMap)
		}
	}

	hasMore := int64(offset+len(rows)) < totalRows

	return &DataFileSummary{
		Format:          format,
		TotalRows:       totalRows,
		TotalCols:       len(allCols),
		FileSize:        int64(len(data)),
		Columns:         allCols,
		SelectedColumns: activeCols,
		Rows:            rows,
		Offset:          offset,
		Limit:           limit,
		HasMore:         hasMore,
	}, nil
}
