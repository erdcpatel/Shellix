package dataview

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ReadExcelData parses an Excel spreadsheet (.xlsx) into tabular rows
func ReadExcelData(data []byte, opts InspectOptions) (*DataFileSummary, error) {
	if int64(len(data)) > MaxDecompressedBytes {
		return nil, fmt.Errorf("excel data exceeds maximum allowed size (%d bytes)", MaxDecompressedBytes)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed opening excel file: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return nil, fmt.Errorf("excel file contains no sheets")
	}

	activeSheet := sheetList[0]

	rowsIter, err := f.Rows(activeSheet)
	if err != nil {
		return nil, fmt.Errorf("failed reading sheet %s: %w", activeSheet, err)
	}

	allCols := make([]ColumnMeta, 0)
	activeCols := make([]string, 0)
	rows := make([]map[string]interface{}, 0)

	if !rowsIter.Next() {
		return &DataFileSummary{
			Format:          "xlsx",
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

	headers, err := rowsIter.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed reading excel headers: %w", err)
	}

	for i, h := range headers {
		clean := strings.TrimSpace(h)
		if clean == "" {
			clean = fmt.Sprintf("col_%d", i+1)
		}
		allCols = append(allCols, ColumnMeta{
			Name:     clean,
			Type:     "EXCEL_CELL",
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

	for rowsIter.Next() {
		record, err := rowsIter.Columns()
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
		Format:          "xlsx",
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
