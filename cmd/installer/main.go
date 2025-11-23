package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ardanlabs/kronk"
	"github.com/hybridgroup/yzma/pkg/download"
)

const (
	libPath     = "libraries"
	versionFile = "version.json"
)

type tag struct {
	TagName string `json:"tag_name"`
}

func main() {
	if err := run(); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}

func run() error {
	currentInstalled, _ := versionInfo(libPath)

	if !currentInstalled {
		if err := kronk.InstallLlama(libPath, download.CPU, true); err != nil {
			return fmt.Errorf("failed to install llama: %q: error: %w", libPath, err)
		}

		f := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			fmt.Println("lib:", path)
			return nil
		}

		if err := filepath.Walk(libPath, f); err != nil {
			return fmt.Errorf("error walking model path: %v", err)
		}
	}

	return nil
}

func versionInfo(libPath string) (bool, error) {
	versionInfoPath := filepath.Join(libPath, versionFile)

	version, err := download.LlamaLatestVersion()
	if err != nil {
		return false, fmt.Errorf("error install: %w", err)
	}

	fmt.Println("Latest Version   :", version)

	d, err := os.ReadFile(versionInfoPath)
	if err != nil {
		return false, fmt.Errorf("error reading version info file: %w", err)
	}

	var tag tag
	if err := json.Unmarshal(d, &tag); err != nil {
		return false, fmt.Errorf("error unmarshalling version info: %w", err)
	}

	fmt.Println("Currently Version:", tag.TagName)

	return version == tag.TagName, nil
}
