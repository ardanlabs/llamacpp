// Package install provides functions for installing and upgrading llama.cpp.
package install

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/hybridgroup/yzma/pkg/download"
)

const versionFile = "version.json"

type tag struct {
	TagName string `json:"tag_name"`
}

// Version provides information about what is installed and what is the
// latest version of llama.cpp available.
type Version struct {
	Latest  string
	Current string
}

// VersionInformation retrieves the version information for the llama.cpp
// libs and the current version installed.
func VersionInformation(libPath string) (Version, error) {
	versionInfoPath := filepath.Join(libPath, versionFile)

	version, err := download.LlamaLatestVersion()
	if err != nil {
		return Version{}, fmt.Errorf("unable to get latest version of llama.cpp: %w", err)
	}

	d, err := os.ReadFile(versionInfoPath)
	if err != nil {
		return Version{Current: "unknown", Latest: version}, nil
	}

	var tag tag
	if err := json.Unmarshal(d, &tag); err != nil {
		return Version{Current: "unknown", Latest: version}, fmt.Errorf("unable to parse version info file: %w", err)
	}

	return Version{Latest: version, Current: tag.TagName}, nil
}

// Libraries installs or upgrades to the latest version of llama.cpp at the
// specified libPath.
func Libraries(libPath string, processor download.Processor, allowUpgrade bool) (Version, error) {
	if err := download.InstallLibraries(libPath, processor, allowUpgrade); err != nil {
		file := filepath.Join(libPath, "libmtmd.dylib")

		if _, err := os.Stat(file); err == nil {
			return Version{}, nil
		}

		return Version{}, fmt.Errorf("unable to install llama.cpp: %w", err)
	}

	return VersionInformation(libPath)
}

// Model installs the model at the specified URL to the specified path.
func Model(modelURL string, modelPath string) (string, error) {
	u, err := url.Parse(modelURL)
	if err != nil {
		return "", fmt.Errorf("unable to parse modelURL: %w", err)
	}

	file := filepath.Join(modelPath, path.Base(u.Path))

	if _, err := os.Stat(file); err == nil {
		return file, nil
	}

	if err := download.GetModel(modelURL, modelPath); err != nil {
		return "", fmt.Errorf("unable to download model: %w", err)
	}

	return file, nil
}
