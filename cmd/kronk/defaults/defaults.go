// Package defaults provides default values for the cli tooling.
package defaults

import (
	"os"
	"path/filepath"
)

// ModelsDir returns the location for the models folder. It will check the
// KRONK_MODELS env var first and then default to the home directory if one
// can be identified. Last resort it will choose the current directory.
func ModelsDir() string {
	if v := os.Getenv("KRONK_MODELS"); v != "" {
		return v
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./kronk/models"
	}

	return filepath.Join(homeDir, "kronk/models")
}
