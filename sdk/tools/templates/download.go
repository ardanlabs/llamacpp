package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	shaFile = ".template_shas.json"
)

// Download retrieves the templates from github.com/ardanlabs/kronk_catalogs.
// Only files modified after the last download are fetched.
func Download(ctx context.Context, basePath string) error {
	if !hasNetwork() {
		return nil
	}

	templatesDir := filepath.Join(basePath, localFolder)
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("creating templates directory: %w", err)
	}

	files, err := listGitHubFolder(ctx, "ardanlabs", "kronk_catalogs", "templates", templatesDir)
	if err != nil {
		return fmt.Errorf("listing templates: %w", err)
	}

	for _, file := range files {
		if err := downloadFile(ctx, templatesDir, file); err != nil {
			return fmt.Errorf("download-template: %w", err)
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

func listGitHubFolder(ctx context.Context, owner string, repo string, path string, templatesDir string) ([]string, error) {
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

	localSHAs := readLocalSHAs(templatesDir)

	var files []string
	for _, item := range items {
		if item.Type != "file" || item.DownloadURL == "" {
			continue
		}
		if localSHAs[item.Name] != item.SHA {
			files = append(files, item.DownloadURL)
		}
	}

	if err := writeLocalSHAs(templatesDir, items); err != nil {
		return nil, fmt.Errorf("writing SHA file: %w", err)
	}

	return files, nil
}

func downloadFile(ctx context.Context, destDir string, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	filePath := filepath.Join(destDir, filepath.Base(url))
	if err := os.WriteFile(filePath, body, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
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
