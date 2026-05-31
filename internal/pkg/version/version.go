package version

// Version is the current semantic version of the application.
const Version = "0.1.0-dev"

// Get returns the current version string.
func Get() string {
	return Version
}
