package tools

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

// FindModelInfo returns file information about a model.
type FindModelInfo struct {
	ModelFile string
	ProjFile  string
}

// FindModel locates the physical location on disk and returns the full path.
func FindModel(modelPath string, modelName string) (FindModelInfo, error) {
	entries, err := os.ReadDir(modelPath)
	if err != nil {
		return FindModelInfo{}, fmt.Errorf("reading models directory: %w", err)
	}

	projName := fmt.Sprintf("mmproj-%s", modelName)

	var fi FindModelInfo

	for _, orgEntry := range entries {
		if !orgEntry.IsDir() {
			continue
		}

		org := orgEntry.Name()

		modelEntries, err := os.ReadDir(fmt.Sprintf("%s/%s", modelPath, org))
		if err != nil {
			continue
		}

		for _, modelEntry := range modelEntries {
			if !modelEntry.IsDir() {
				continue
			}
			model := modelEntry.Name()

			fileEntries, err := os.ReadDir(fmt.Sprintf("%s/%s/%s", modelPath, org, model))
			if err != nil {
				continue
			}

			for _, fileEntry := range fileEntries {
				if fileEntry.IsDir() {
					continue
				}

				if fileEntry.Name() == ".DS_Store" {
					continue
				}

				if fileEntry.Name() == modelName {
					fi.ModelFile = filepath.Join(modelPath, org, model, fileEntry.Name())
					continue
				}

				if fileEntry.Name() == projName {
					fi.ProjFile = filepath.Join(modelPath, org, model, fileEntry.Name())
					continue
				}
			}
		}
	}

	if fi.ModelFile == "" {
		return FindModelInfo{}, fmt.Errorf("model %q not found", modelName)
	}

	return fi, nil
}

func MustFindModel(modelPath string, modelName string) FindModelInfo {
	fi, err := FindModel(modelPath, modelName)
	if err != nil {
		panic(err.Error())
	}

	return fi
}

// =============================================================================

// DownloadModelInfo provides information about the models that were downloaded.
type DownloadModelInfo struct {
	ModelFile  string
	ProjFile   string
	Downloaded bool
}

// DownloadModel performs a complete workflow for downloading and installing
// the specified model.
func DownloadModel(ctx context.Context, log Logger, modelURL string, projURL string, modelPath string) (DownloadModelInfo, error) {
	u, _ := url.Parse(modelURL)
	filename := path.Base(u.Path)
	name := strings.TrimSuffix(filename, path.Ext(filename))
	log(ctx, "download-model", "status", "check model installation", "model-path", modelPath, "model-url", modelURL, "proj-url", projURL, "model-name", name)

	f := func(src string, currentSize int64, totalSize int64, mibPerSec float64, complete bool) {
		log(ctx, fmt.Sprintf("\x1b[1A\r\x1b[KDownloading %s... %d MiB of %d MiB (%.2f MiB/s)", src, currentSize/(1024*1024), totalSize/(1024*1024), mibPerSec))
		if complete {
			log(ctx, "download complete")
		}
	}

	info, err := downloadModel(modelURL, projURL, modelPath, f)
	if err != nil {
		return DownloadModelInfo{}, fmt.Errorf("unable to download model: %w", err)
	}

	switch info.Downloaded {
	case true:
		log(ctx, "download-model", "status", "model downloaded", "model-file", info.ModelFile, "proj-file", info.ProjFile)

	default:
		log(ctx, "download-model", "status", "model already existed", "model-file", info.ModelFile, "proj-file", info.ProjFile)
	}

	return info, nil
}

func downloadModel(modelURL string, projURL string, modelPath string, progress ProgressFunc) (DownloadModelInfo, error) {
	modelFile, downloadedMF, err := pullModel(modelURL, modelPath, progress)
	if err != nil {
		return DownloadModelInfo{}, err
	}

	if projURL == "" {
		return DownloadModelInfo{ModelFile: modelFile, Downloaded: downloadedMF}, nil
	}

	modelFileName := filepath.Base(modelFile)
	profFileName := fmt.Sprintf("mmproj-%s", modelFileName)
	newProjFile := strings.Replace(modelFile, modelFileName, profFileName, 1)

	if _, err := os.Stat(newProjFile); err == nil {
		inf := DownloadModelInfo{
			ModelFile:  modelFile,
			ProjFile:   newProjFile,
			Downloaded: downloadedMF || false,
		}

		return inf, nil
	}

	projFile, downloadedPF, err := pullModel(projURL, modelPath, progress)
	if err != nil {
		return DownloadModelInfo{}, err
	}

	if err := os.Rename(projFile, newProjFile); err != nil {
		return DownloadModelInfo{}, fmt.Errorf("unable to rename projector file: %w", err)
	}

	inf := DownloadModelInfo{
		ModelFile:  modelFile,
		ProjFile:   newProjFile,
		Downloaded: downloadedMF || downloadedPF,
	}

	return inf, nil
}

func pullModel(fileURL string, filePath string, progress ProgressFunc) (string, bool, error) {
	mURL, err := url.Parse(fileURL)
	if err != nil {
		return "", false, fmt.Errorf("unable to parse fileURL: %w", err)
	}

	parts := strings.Split(mURL.Path, "/")
	if len(parts) < 3 {
		return "", false, fmt.Errorf("invalid huggingface url: %q", mURL.Path)
	}

	filePath = filepath.Join(filePath, parts[1], parts[2])
	mFile := filepath.Join(filePath, path.Base(mURL.Path))

	// The downloader can check if we have the full file and if it's of the
	// correct size. If we are not given a progress function, we can't check
	// the file size and the existence of the file is all we can do not to
	// start a download.
	if progress == nil {
		if _, err := os.Stat(mFile); err == nil {
			return mFile, false, nil
		}
	}

	downloaded, err := DownloadFile(context.Background(), fileURL, filePath, progress)
	if err != nil {
		return "", false, fmt.Errorf("unable to download model: %w", err)
	}

	return mFile, downloaded, nil
}

// =============================================================================

// ListModelInfo provides information about a model.
type ListModelInfo struct {
	Organization string
	ModelName    string
	ModelFile    string
	Size         int64
	Modified     time.Time
}

// ListModels lists all the models in the given directory.
func ListModels(modelPath string) ([]ListModelInfo, error) {
	entries, err := os.ReadDir(modelPath)
	if err != nil {
		return nil, fmt.Errorf("reading models directory: %w", err)
	}

	var list []ListModelInfo

	for _, orgEntry := range entries {
		if !orgEntry.IsDir() {
			continue
		}

		org := orgEntry.Name()

		modelEntries, err := os.ReadDir(fmt.Sprintf("%s/%s", modelPath, org))
		if err != nil {
			continue
		}

		for _, modelEntry := range modelEntries {
			if !modelEntry.IsDir() {
				continue
			}
			model := modelEntry.Name()

			fileEntries, err := os.ReadDir(fmt.Sprintf("%s/%s/%s", modelPath, org, model))
			if err != nil {
				continue
			}

			for _, fileEntry := range fileEntries {
				if fileEntry.IsDir() {
					continue
				}

				if fileEntry.Name() == ".DS_Store" {
					continue
				}

				info, err := fileEntry.Info()
				if err != nil {
					continue
				}

				list = append(list, ListModelInfo{
					Organization: org,
					ModelName:    model,
					ModelFile:    fileEntry.Name(),
					Size:         info.Size(),
					Modified:     info.ModTime(),
				})
			}
		}
	}

	return list, nil
}

func ListModelsFmt(models []ListModelInfo) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ORG\tMODEL\tFILE\tSIZE\tMODIFIED")

	for _, model := range models {
		size := formatSize(model.Size)
		modified := formatTime(model.Modified)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", model.Organization, model.ModelName, model.ModelFile, size, modified)
	}

	w.Flush()
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}
