package jsonview

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Matches array index notation like "items[2]" -> ("items", "2")
	arrayIndexRegex = regexp.MustCompile(`^([^\[]+)\[(\d+|\*)\]$`)
	bracketOnlyRegex = regexp.MustCompile(`^\[(\d+|\*)\]$`)
)

// QueryFromReader decodes JSON from an io.Reader and evaluates the given path expression
func QueryFromReader(r io.Reader, path string) (interface{}, error) {
	var root interface{}
	dec := json.NewDecoder(r)
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}
	return EvaluateJSONPath(root, path)
}

// EvaluateJSONPath navigates a decoded JSON data structure according to path expression
func EvaluateJSONPath(data interface{}, path string) (interface{}, error) {
	cleanPath := strings.TrimSpace(path)
	cleanPath = strings.TrimPrefix(cleanPath, "$")
	cleanPath = strings.TrimPrefix(cleanPath, ".")

	if cleanPath == "" || cleanPath == "." {
		return data, nil
	}

	tokens := parsePathTokens(cleanPath)
	current := data

	for _, token := range tokens {
		if current == nil {
			return nil, nil
		}

		switch tok := token.(type) {
		case string:
			switch m := current.(type) {
			case map[string]interface{}:
				val, exists := m[tok]
				if !exists {
					return nil, fmt.Errorf("key %q not found", tok)
				}
				current = val
			default:
				return nil, fmt.Errorf("cannot access field %q on non-object (%T)", tok, current)
			}

		case int:
			switch s := current.(type) {
			case []interface{}:
				if tok < 0 || tok >= len(s) {
					return nil, fmt.Errorf("index %d out of bounds (length %d)", tok, len(s))
				}
				current = s[tok]
			default:
				return nil, fmt.Errorf("cannot index non-array (%T) with %d", current, tok)
			}

		case wildcardToken:
			switch s := current.(type) {
			case []interface{}:
				// Wildcard returns a slice of whatever follows
				// Handled by remaining tokens if any
				return handleWildcard(s, tokens[indexOfToken(tokens, tok)+1:])
			default:
				return nil, fmt.Errorf("wildcard [*] not supported on %T", current)
			}
		}
	}

	return current, nil
}

type wildcardToken struct{}

func indexOfToken(tokens []interface{}, target interface{}) int {
	for i, t := range tokens {
		if t == target {
			return i
		}
	}
	return -1
}

func handleWildcard(items []interface{}, remainingTokens []interface{}) ([]interface{}, error) {
	if len(remainingTokens) == 0 {
		return items, nil
	}

	results := make([]interface{}, 0, len(items))
	for _, item := range items {
		subCurrent := item
		for _, token := range remainingTokens {
			if subCurrent == nil {
				break
			}
			switch tok := token.(type) {
			case string:
				if m, ok := subCurrent.(map[string]interface{}); ok {
					subCurrent = m[tok]
				} else {
					subCurrent = nil
				}
			case int:
				if s, ok := subCurrent.([]interface{}); ok && tok >= 0 && tok < len(s) {
					subCurrent = s[tok]
				} else {
					subCurrent = nil
				}
			}
		}
		if subCurrent != nil {
			results = append(results, subCurrent)
		}
	}
	return results, nil
}

func parsePathTokens(path string) []interface{} {
	var tokens []interface{}
	parts := strings.Split(path, ".")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if part contains bracket notation: key[0] or [0]
		for part != "" {
			if bracketOnlyRegex.MatchString(part) {
				m := bracketOnlyRegex.FindStringSubmatch(part)
				if m[1] == "*" {
					tokens = append(tokens, wildcardToken{})
				} else {
					idx, _ := strconv.Atoi(m[1])
					tokens = append(tokens, idx)
				}
				break
			}

			if arrayIndexRegex.MatchString(part) {
				m := arrayIndexRegex.FindStringSubmatch(part)
				tokens = append(tokens, m[1])
				if m[2] == "*" {
					tokens = append(tokens, wildcardToken{})
				} else {
					idx, _ := strconv.Atoi(m[2])
					tokens = append(tokens, idx)
				}
				break
			}

			// Check if bracket is within part e.g. "users[0][1]"
			idxBracket := strings.Index(part, "[")
			if idxBracket != -1 {
				key := part[:idxBracket]
				if key != "" {
					tokens = append(tokens, key)
				}
				part = part[idxBracket:]
				continue
			}

			tokens = append(tokens, part)
			break
		}
	}

	return tokens
}

// FormatQueryResult serializes the query output cleanly
func FormatQueryResult(val interface{}, pretty bool) (string, error) {
	if str, ok := val.(string); ok {
		return str, nil
	}
	if val == nil {
		return "null", nil
	}

	if pretty {
		b, err := json.MarshalIndent(val, "", "  ")
		return string(b), err
	}
	b, err := json.Marshal(val)
	return string(b), err
}
