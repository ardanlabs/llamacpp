package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	timestampFile   = ".last_download"
	timestampFormat = time.RFC3339
)

// Download retrieves the templates from github.com/ardanlabs/kronk_catalogs.
// Only files modified after the last download are fetched.
func Download(ctx context.Context, basePath string) error {
	templatesDir := filepath.Join(basePath, localFolder)
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("creating templates directory: %w", err)
	}

	lastDownload := readLastDownloadTime(templatesDir)

	files, err := listGitHubFolder(ctx, "ardanlabs", "kronk_catalogs", "templates", lastDownload)
	if err != nil {
		return fmt.Errorf("listing templates: %w", err)
	}

	for _, file := range files {
		if err := downloadFile(ctx, templatesDir, file); err != nil {
			return fmt.Errorf("download-template: %w", err)
		}
	}

	if err := writeLastDownloadTime(templatesDir); err != nil {
		return fmt.Errorf("writing timestamp: %w", err)
	}

	return nil
}

// =============================================================================

func listGitHubFolder(ctx context.Context, owner string, repo string, path string, since time.Time) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?path=%s&per_page=1", owner, repo, path)
	if !since.IsZero() {
		url += "&since=" + since.Format(time.RFC3339)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching commits: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var commits []struct{}
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, fmt.Errorf("decoding commits: %w", err)
	}

	if len(commits) == 0 && !since.IsZero() {
		return nil, nil
	}

	return listGitHubContents(ctx, owner, repo, path)
}

func listGitHubContents(ctx context.Context, owner string, repo string, path string) ([]string, error) {
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

	var items []struct {
		Type        string `json:"type"`
		DownloadURL string `json:"download_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var files []string
	for _, item := range items {
		if item.Type == "file" && item.DownloadURL != "" {
			files = append(files, item.DownloadURL)
		}
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

func readLastDownloadTime(dir string) time.Time {
	data, err := os.ReadFile(filepath.Join(dir, timestampFile))
	if err != nil {
		return time.Time{}
	}

	t, err := time.Parse(timestampFormat, string(data))
	if err != nil {
		return time.Time{}
	}

	return t
}

func writeLastDownloadTime(dir string) error {
	ts := time.Now().UTC().Format(timestampFormat)
	return os.WriteFile(filepath.Join(dir, timestampFile), []byte(ts), 0644)
}
