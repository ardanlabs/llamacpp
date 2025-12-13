package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v2"
)

// Retrieve reads the catalog previously downloaded.
func Retrieve(basePath string) ([]Catalog, error) {
	catalogDir := filepath.Join(basePath, localFolder)

	entries, err := os.ReadDir(catalogDir)
	if err != nil {
		return nil, fmt.Errorf("read catalog dir: %w", err)
	}

	var catalogs []Catalog

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		filePath := filepath.Join(catalogDir, entry.Name())

		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", entry.Name(), err)
		}

		var catalog Catalog
		if err := yaml.Unmarshal(data, &catalog); err != nil {
			return nil, fmt.Errorf("unmarshal %s: %w", entry.Name(), err)
		}

		catalogs = append(catalogs, catalog)
	}

	return catalogs, nil
}
