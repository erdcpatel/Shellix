package shortcuts

import "time"

// Bookmark represents a saved filesystem directory path
type Bookmark struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
}

// Snippet represents a reusable shell command
type Snippet struct {
	Name        string    `json:"name"`
	Command     string    `json:"command"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// DataStore serializes bookmarks and snippets to disk
type DataStore struct {
	Bookmarks map[string]Bookmark `json:"bookmarks"`
	Snippets  map[string]Snippet  `json:"snippets"`
}
