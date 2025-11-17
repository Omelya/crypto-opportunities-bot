package version

// Version represents the application version
// This can be overridden at build time using ldflags:
// go build -ldflags "-X crypto-opportunities-bot/internal/version.Version=v1.2.3"
var Version = "dev"

// BuildTime represents when the binary was built
var BuildTime = "unknown"

// GitCommit represents the git commit hash
var GitCommit = "unknown"

// GetVersion returns the current application version
func GetVersion() string {
	if Version == "" {
		return "dev"
	}
	return Version
}

// GetBuildInfo returns complete build information
func GetBuildInfo() map[string]string {
	return map[string]string{
		"version":    GetVersion(),
		"build_time": BuildTime,
		"git_commit": GitCommit,
	}
}
