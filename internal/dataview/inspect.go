package dataview

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

	// Ensure slices are non-nil so they serialize to [] instead of null in JSON
	if summary.Columns == nil {
		summary.Columns = make([]ColumnMeta, 0)
	}
	if summary.SelectedColumns == nil {
		summary.SelectedColumns = make([]string, 0)
	}
	if summary.Rows == nil {
		summary.Rows = make([]map[string]interface{}, 0)
	}

	return summary, nil
}

// NormalizeValue recursively converts byte slices to strings and normalizes nested structures
func NormalizeValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []byte:
		return string(val)
	case string:
		return val
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return val
	case map[string]interface{}:
		res := make(map[string]interface{}, len(val))
		for k, item := range val {
			res[k] = NormalizeValue(item)
		}
		return res
	case map[interface{}]interface{}:
		res := make(map[string]interface{}, len(val))
		for k, item := range val {
			res[fmt.Sprintf("%v", k)] = NormalizeValue(item)
		}
		return res
	case []interface{}:
		res := make([]interface{}, len(val))
		for i, item := range val {
			res[i] = NormalizeValue(item)
		}
		return res
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return nil
		}
		return NormalizeValue(rv.Elem().Interface())
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return string(rv.Bytes())
		}
		res := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			res[i] = NormalizeValue(rv.Index(i).Interface())
		}
		return res
	case reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			b := make([]byte, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				b[i] = byte(rv.Index(i).Uint())
			}
			return string(b)
		}
		res := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			res[i] = NormalizeValue(rv.Index(i).Interface())
		}
		return res
	case reflect.Map:
		res := make(map[string]interface{}, rv.Len())
		for _, key := range rv.MapKeys() {
			kStr := fmt.Sprintf("%v", key.Interface())
			res[kStr] = NormalizeValue(rv.MapIndex(key).Interface())
		}
		return res
	case reflect.Struct:
		res := make(map[string]interface{})
		typ := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			field := typ.Field(i)
			if field.PkgPath != "" { // unexported field
				continue
			}
			name := field.Name
			if tag := field.Tag.Get("json"); tag != "" {
				parts := strings.Split(tag, ",")
				if parts[0] == "-" {
					continue
				}
				if parts[0] != "" {
					name = parts[0]
				}
			}
			res[name] = NormalizeValue(rv.Field(i).Interface())
		}
		return res
	default:
		return v
	}
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
			val, exists := row[h]
			if (!exists || val == nil) && strings.Contains(h, ".") {
				parts := strings.Split(h, ".")
				var cur interface{} = row
				found := true
				for _, p := range parts {
					if m, ok := cur.(map[string]interface{}); ok {
						cur = m[p]
					} else {
						found = false
						break
					}
				}
				if found && cur != nil {
					val = cur
				}
			}
			if val == nil {
				record = append(record, "")
			} else {
				record = append(record, fmt.Sprintf("%v", val))
			}
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}
