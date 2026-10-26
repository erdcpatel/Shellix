package dataview

// ColumnMeta describes a column in a tabular data file
type ColumnMeta struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// DataFileSummary contains complete inspected metadata and sample/window rows
type DataFileSummary struct {
	FileName        string                   `json:"file_name"`
	Format          string                   `json:"format"`      // "parquet", "csv", "tsv", "xlsx", "jsonl"
	Compression     string                   `json:"compression"` // "none", "gzip", "nested-gzip"
	TotalRows       int64                    `json:"total_rows"`
	TotalCols       int                      `json:"total_cols"`
	FileSize        int64                    `json:"file_size"`
	Columns         []ColumnMeta             `json:"columns"`
	SelectedColumns []string                 `json:"selected_columns"`
	Rows            []map[string]interface{} `json:"rows"`
	Offset          int                      `json:"offset"`
	Limit           int                      `json:"limit"`
	HasMore         bool                     `json:"has_more"`
}

// InspectOptions parameters for querying and projecting tabular files
type InspectOptions struct {
	Columns []string `json:"columns"` // Column projection
	Offset  int      `json:"offset"`  // Row offset
	Limit   int      `json:"limit"`   // Max rows to fetch (default: 50)
	Filter  string   `json:"filter"`  // Substring or regex filter
}
