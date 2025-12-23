package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.yaml.in/yaml/v2"
)

const (
	shaFile = ".catalog_shas.json"
)

// Download retrieves the catalog from github.com/ardanlabs/kronk_catalogs.
// Only files modified after the last download are fetched.
func Download(ctx context.Context, basePath string) error {
	if !hasNetwork() {
		return nil
	}

	catalogDir := filepath.Join(basePath, localFolder)
	if err := os.MkdirAll(catalogDir, 0755); err != nil {
		return fmt.Errorf("creating catalogs directory: %w", err)
	}

	files, err := listGitHubFolder(ctx, "ardanlabs", "kronk_catalogs", "catalogs", catalogDir)
	if err != nil {
		return fmt.Errorf("listing catalogs: %w", err)
	}

	for _, file := range files {
		if err := downloadCatalog(ctx, catalogDir, file); err != nil {
			return fmt.Errorf("download-catalog: %w", err)
		}
	}

	if len(files) > 0 {
		if err := buildIndex(basePath); err != nil {
			return fmt.Errorf("build-index: %w", err)
		}
	}

	return nil
}

// =============================================================================

type gitHubFile struct {
	Name        string `json:"name"`
	SHA         string `json:"sha"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

func listGitHubFolder(ctx context.Context, owner string, repo string, path string, catalogDir string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching folder listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var items []gitHubFile
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	localSHAs := readLocalSHAs(catalogDir)

	var files []string
	for _, item := range items {
		if item.Type != "file" || item.DownloadURL == "" {
			continue
		}
		if localSHAs[item.Name] != item.SHA {
			files = append(files, item.DownloadURL)
		}
	}

	if err := writeLocalSHAs(catalogDir, items); err != nil {
		return nil, fmt.Errorf("writing SHA file: %w", err)
	}

	return files, nil
}

func downloadCatalog(ctx context.Context, catalogDir string, url string) error {
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

	filePath := filepath.Join(catalogDir, filepath.Base(url))
	if err := os.WriteFile(filePath, body, 0644); err != nil {
		return fmt.Errorf("writing catalog file: %w", err)
	}

	return nil
}

func readLocalSHAs(dir string) map[string]string {
	data, err := os.ReadFile(filepath.Join(dir, shaFile))
	if err != nil {
		return make(map[string]string)
	}

	var shas map[string]string
	if err := json.Unmarshal(data, &shas); err != nil {
		return make(map[string]string)
	}

	return shas
}

func writeLocalSHAs(dir string, items []gitHubFile) error {
	shas := make(map[string]string)
	for _, item := range items {
		if item.Type == "file" {
			shas[item.Name] = item.SHA
		}
	}

	data, err := json.Marshal(shas)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, shaFile), data, 0644)
}

var biMutex sync.Mutex

func buildIndex(basePath string) error {
	biMutex.Lock()
	defer biMutex.Unlock()

	catalogDir := filepath.Join(basePath, localFolder)

	entries, err := os.ReadDir(catalogDir)
	if err != nil {
		return fmt.Errorf("read catalog dir: %w", err)
	}

	index := make(map[string]string)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		if entry.Name() == indexFile {
			continue
		}

		filePath := filepath.Join(catalogDir, entry.Name())

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read file %s: %w", entry.Name(), err)
		}

		var catalog Catalog
		if err := yaml.Unmarshal(data, &catalog); err != nil {
			return fmt.Errorf("unmarshal %s: %w", entry.Name(), err)
		}

		for _, model := range catalog.Models {
			modelID := strings.ToLower(model.ID)
			index[modelID] = entry.Name()
		}
	}

	indexData, err := yaml.Marshal(&index)
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	indexPath := filepath.Join(catalogDir, indexFile)
	if err := os.WriteFile(indexPath, indexData, 0644); err != nil {
		return fmt.Errorf("write index file: %w", err)
	}

	return nil
}
