package templates

import (
	"fmt"
	"os"
	"path/filepath"
)

// RetrieveFile reads the content of a specific file from the local templates folder.
func RetrieveFile(basePath string, fileName string) (string, error) {
	filePath := filepath.Join(basePath, localFolder, fileName)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading template file: %w", err)
	}

	return string(content), nil
}
