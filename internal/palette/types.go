package palette

// ActionCategory classifies workbench commands in the palette
type ActionCategory string

const (
	CatCore     ActionCategory = "CORE"
	CatFlow     ActionCategory = "FLOW"
	CatSSH      ActionCategory = "SSH"
	CatBookmark ActionCategory = "BOOKMARK"
	CatSnippet  ActionCategory = "SNIP"
	CatHistory  ActionCategory = "HIST"
)

// PaletteAction represents a runnable command or navigation item
type PaletteAction struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Category    ActionCategory `json:"category"`
	Command     string         `json:"command"`
}
