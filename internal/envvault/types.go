package envvault

import (
	"time"
)

// EnvVaultData contains the decrypted key-value environment variables
type EnvVaultData struct {
	Version   int               `json:"version"`
	Project   string            `json:"project"`
	Variables map[string]string `json:"variables"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// EnvVarSummary summarizes a variable for display
type EnvVarSummary struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Masked   string `json:"masked"`
	IsSecret bool   `json:"is_secret"`
}
