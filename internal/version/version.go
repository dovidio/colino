package version

import "fmt"

const (
	// Version is the current version of Colino
	Version = "0.2.0"
)

// GetVersion returns the current version string
func GetVersion() string {
	return fmt.Sprintf("Colino %s", Version)
}