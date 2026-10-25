package ssh

import "time"

// HostProfile defines a remote Linux/Unix server target
type HostProfile struct {
	Name        string    `json:"name"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	User        string    `json:"user"`
	KeyPath     string    `json:"key_path,omitempty"`
	Password    string    `json:"password,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// HostStore persists server profiles
type HostStore struct {
	Hosts map[string]HostProfile `json:"hosts"`
}

// RemoteExecResult contains the result of a remote command
type RemoteExecResult struct {
	HostName string
	Output   string
	Duration time.Duration
	Success  bool
	Error    error
}
