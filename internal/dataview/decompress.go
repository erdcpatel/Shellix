package dataview

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	// MaxDecompressedBytes caps the decompressed stream (256 MB) to prevent zip/gzip bomb OOM attacks
	MaxDecompressedBytes = 256 * 1024 * 1024
	// MaxDecompressLayers limits nested decompression recursion
	MaxDecompressLayers = 5
)

// ErrDecompressionBomb is returned when decompressed data exceeds the safety limit
var ErrDecompressionBomb = errors.New("decompression limit exceeded (potential decompression bomb)")

// DecompressResult wraps the decompressed stream and detection metadata
type DecompressResult struct {
	Data        []byte
	Compression string // "none", "gzip", "nested-gzip"
	Format      string // "parquet", "csv", "tsv", "xlsx", "unknown"
}

// ReadAndDecompress reads a file or raw bytes and unwraps any nested gzip compression layers up to MaxDecompressedBytes
func ReadAndDecompress(data []byte) (*DecompressResult, error) {
	return ReadAndDecompressWithLimit(data, MaxDecompressedBytes)
}

// ReadAndDecompressWithLimit reads a file or raw bytes and unwraps nested gzip compression with a custom byte limit
func ReadAndDecompressWithLimit(data []byte, maxBytes int64) (*DecompressResult, error) {
	if int64(len(data)) > maxBytes {
		return nil, ErrDecompressionBomb
	}
	currentData := data
	layers := 0

	// Loop to recursively unwrap multi-level gzip (e.g. .gz or .gz.gz)
	for {
		if len(currentData) < 2 {
			break
		}
		// Gzip magic signature 0x1F 0x8B
		if currentData[0] == 0x1F && currentData[1] == 0x8B {
			if layers >= MaxDecompressLayers {
				return nil, fmt.Errorf("exceeded maximum nested decompression layers (%d)", MaxDecompressLayers)
			}
			gzReader, err := gzip.NewReader(bytes.NewReader(currentData))
			if err != nil {
				break
			}
			// Use LimitReader to guard against decompression bombs
			limited := io.LimitReader(gzReader, maxBytes+1)
			uncompressed, err := io.ReadAll(limited)
			_ = gzReader.Close()
			if err != nil {
				return nil, fmt.Errorf("decompression stream error: %w", err)
			}
			if int64(len(uncompressed)) > maxBytes {
				return nil, ErrDecompressionBomb
			}
			currentData = uncompressed
			layers++
		} else {
			break
		}
	}

	compression := "none"
	if layers == 1 {
		compression = "gzip"
	} else if layers > 1 {
		compression = fmt.Sprintf("nested-gzip (%d layers)", layers)
	}

	format := detectFormat(currentData)

	return &DecompressResult{
		Data:        currentData,
		Compression: compression,
		Format:      format,
	}, nil
}

// ReadFileAndDecompress opens a file and recursively unwraps any compression
func ReadFileAndDecompress(filePath string) (*DecompressResult, error) {
	rawBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed reading file %s: %w", filePath, err)
	}
	return ReadAndDecompress(rawBytes)
}

func detectFormat(data []byte) string {
	if len(data) >= 4 {
		// Parquet magic header 'PAR1'
		if data[0] == 'P' && data[1] == 'A' && data[2] == 'R' && data[3] == '1' {
			return "parquet"
		}
		// Parquet magic footer 'PAR1'
		if len(data) > 8 {
			last4 := data[len(data)-4:]
			if last4[0] == 'P' && last4[1] == 'A' && last4[2] == 'R' && last4[3] == '1' {
				return "parquet"
			}
		}
		// ZIP / XLSX magic 'PK\x03\x04'
		if data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04 {
			return "xlsx"
		}
	}

	// Sniff text for CSV / TSV / JSON
	sampleLen := 1024
	if len(data) < sampleLen {
		sampleLen = len(data)
	}
	sample := string(data[:sampleLen])

	tabCount := 0
	commaCount := 0
	for _, ch := range sample {
		if ch == '\t' {
			tabCount++
		} else if ch == ',' {
			commaCount++
		}
	}

	if tabCount > commaCount && tabCount > 2 {
		return "tsv"
	}
	if commaCount > 2 {
		return "csv"
	}

	return "csv" // Default tabular fallback
}
