package java

// BuildToolType identifies detected Java build systems
type BuildToolType string

const (
	BuildToolMaven      BuildToolType = "maven"
	BuildToolGradle     BuildToolType = "gradle"
	BuildToolSingleFile BuildToolType = "single-file"
	BuildToolUnknown    BuildToolType = "unknown"
)

// EnvironmentInfo encapsulates detected Java toolchain metadata
type EnvironmentInfo struct {
	JavaHome        string
	JavaVersion     string
	JavacVersion    string
	BuildTool       BuildToolType
	BuildFile       string
	WrapperBinary   string
	SystemBinary    string
	HasWrapper      bool
	SourceFilesCount int
}
