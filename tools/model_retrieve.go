package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ardanlabs/kronk"
	"github.com/ardanlabs/kronk/model"
)

// ModelFile provides information about a model.
type ModelFile struct {
	ID          string
	OwnedBy     string
	ModelFamily string
	Size        int64
	Modified    time.Time
}

// RetrieveModelFiles returns all the models in the given model directory.
func RetrieveModelFiles(modelBasePath string) ([]ModelFile, error) {
	entries, err := os.ReadDir(modelBasePath)
	if err != nil {
		return nil, fmt.Errorf("list-models: reading models directory: %w", err)
	}

	var list []ModelFile

	for _, orgEntry := range entries {
		if !orgEntry.IsDir() {
			continue
		}

		org := orgEntry.Name()

		modelEntries, err := os.ReadDir(fmt.Sprintf("%s/%s", modelBasePath, org))
		if err != nil {
			continue
		}

		for _, modelEntry := range modelEntries {
			if !modelEntry.IsDir() {
				continue
			}
			modelFamily := modelEntry.Name()

			fileEntries, err := os.ReadDir(fmt.Sprintf("%s/%s/%s", modelBasePath, org, modelFamily))
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

				if strings.HasPrefix(fileEntry.Name(), "mmproj") {
					continue
				}

				info, err := fileEntry.Info()
				if err != nil {
					continue
				}

				list = append(list, ModelFile{
					ID:          extractModelID(fileEntry.Name()),
					OwnedBy:     org,
					ModelFamily: modelFamily,
					Size:        info.Size(),
					Modified:    info.ModTime(),
				})
			}
		}
	}

	slices.SortFunc(list, func(a, b ModelFile) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})

	return list, nil
}

// =============================================================================

// ModelInfo provides all the model details.
type ModelInfo struct {
	ID      string
	Object  string
	Created int64
	OwnedBy string
	Details model.ModelInfo
}

// RetrieveModelInfo provides details for the specified model.
func RetrieveModelInfo(libPath string, modelBasePath string, modelID string) (ModelInfo, error) {
	modelID = strings.ToLower(modelID)

	fi, err := RetrieveModelPath(modelBasePath, modelID)
	if err != nil {
		return ModelInfo{}, err
	}

	if err := kronk.Init(libPath, kronk.LogSilent); err != nil {
		return ModelInfo{}, fmt.Errorf("show-model: unable to init kronk: %w", err)
	}

	const modelInstances = 1
	krn, err := kronk.New(modelInstances, model.Config{
		ModelFile:      fi.ModelFile,
		ProjectionFile: fi.ProjFile,
	})

	if err != nil {
		return ModelInfo{}, fmt.Errorf("show-model: unable to load kronk: %w", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		krn.Unload(ctx)
	}()

	models, err := RetrieveModelFiles(modelBasePath)
	if err != nil {
		return ModelInfo{}, fmt.Errorf("show-model: unable to get model file information: %w", err)
	}

	var modelFile ModelFile
	for _, model := range models {
		id := strings.ToLower(model.ID)
		if id == modelID {
			modelFile = model
			break
		}
	}

	mi := ModelInfo{
		ID:      modelFile.ID,
		Object:  "model",
		Created: modelFile.Modified.UnixMilli(),
		OwnedBy: modelFile.OwnedBy,
		Details: krn.ModelInfo(),
	}

	return mi, nil
}

// =============================================================================

// ModelPath returns file path information about a model.
type ModelPath struct {
	ModelFile  string
	ProjFile   string
	Downloaded bool
}

// RetrieveModelPath locates the physical location on disk and returns the full path.
func RetrieveModelPath(modelBasePath string, modelID string) (ModelPath, error) {
	entries, err := os.ReadDir(modelBasePath)
	if err != nil {
		return ModelPath{}, fmt.Errorf("find-model: reading models directory: %w", err)
	}

	projID := fmt.Sprintf("mmproj-%s", modelID)

	modelID = strings.ToLower(modelID)
	projID = strings.ToLower(projID)

	var fi ModelPath

	for _, orgEntry := range entries {
		if !orgEntry.IsDir() {
			continue
		}

		org := orgEntry.Name()

		modelEntries, err := os.ReadDir(fmt.Sprintf("%s/%s", modelBasePath, org))
		if err != nil {
			continue
		}

		for _, modelEntry := range modelEntries {
			if !modelEntry.IsDir() {
				continue
			}
			model := modelEntry.Name()

			fileEntries, err := os.ReadDir(fmt.Sprintf("%s/%s/%s", modelBasePath, org, model))
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

				id := strings.ToLower(strings.TrimSuffix(fileEntry.Name(), filepath.Ext(fileEntry.Name())))

				if id == modelID {
					fi.ModelFile = filepath.Join(modelBasePath, org, model, fileEntry.Name())
					continue
				}

				if id == projID {
					fi.ProjFile = filepath.Join(modelBasePath, org, model, fileEntry.Name())
					continue
				}
			}
		}
	}

	if fi.ModelFile == "" {
		return ModelPath{}, fmt.Errorf("find-model: model id %q not found", modelID)
	}

	return fi, nil
}

// MustRetrieveModel finds a model and panics if the model was not found. This
// should only be used for testing.
func MustRetrieveModel(modelBasePath string, modelID string) ModelPath {
	fi, err := RetrieveModelPath(modelBasePath, modelID)
	if err != nil {
		panic(err.Error())
	}

	return fi
}
