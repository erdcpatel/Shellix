package jsonview

// ValueDiff represents a structural difference in values at a given JSONPath
type ValueDiff struct {
	Path string      `json:"path"`
	Val1 interface{} `json:"val1"`
	Val2 interface{} `json:"val2"`
}

// JSONDiffResult contains detailed structural comparison between two JSON documents
type JSONDiffResult struct {
	File1           string      `json:"file1"`
	File2           string      `json:"file2"`
	Size1           int64       `json:"size1"`
	Size2           int64       `json:"size2"`
	Identical       bool        `json:"identical"`
	MatchPercentage float64     `json:"match_percentage"`
	AddedKeys       []string    `json:"added_keys"`
	RemovedKeys     []string    `json:"removed_keys"`
	ChangedValues   []ValueDiff `json:"changed_values"`
}

// ViewerConfig options for high-throughput streaming file viewing
type ViewerConfig struct {
	FilePath        string
	MaxLines        int
	Tail            bool
	Follow          bool
	ShowLineNumbers bool
	SearchPattern   string
}
