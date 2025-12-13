package catalog

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const localFolder = "catalogs"

var files = []string{
	"https://raw.githubusercontent.com/ardanlabs/kronk_catalogs/refs/heads/main/catalogs/audio_text_to_text.yaml",
	"https://raw.githubusercontent.com/ardanlabs/kronk_catalogs/refs/heads/main/catalogs/embedding.yaml",
	"https://raw.githubusercontent.com/ardanlabs/kronk_catalogs/refs/heads/main/catalogs/image_text_to_text.yaml",
	"https://raw.githubusercontent.com/ardanlabs/kronk_catalogs/refs/heads/main/catalogs/text_generation.yaml",
}

// Download retrieves the catalog from github.com/ardanlabs/kronk_catalogs.
func Download(ctx context.Context, basePath string) error {
	for _, file := range files {
		if err := downloadCatalog(ctx, basePath, file); err != nil {
			return fmt.Errorf("download-catalog: %w", err)
		}
	}

	return nil
}

func downloadCatalog(ctx context.Context, basePath string, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	catalogDir := filepath.Join(basePath, localFolder)
	if err := os.MkdirAll(catalogDir, 0755); err != nil {
		return fmt.Errorf("creating catalogs directory: %w", err)
	}

	filePath := filepath.Join(catalogDir, filepath.Base(url))
	if err := os.WriteFile(filePath, body, 0644); err != nil {
		return fmt.Errorf("writing catalog file: %w", err)
	}

	return nil
}
