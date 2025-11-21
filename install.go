package kronk

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/hybridgroup/yzma/pkg/download"
)

func InstallLlama(libPath string, processor download.Processor, allowUpgrade bool) error {
	if err := download.InstallLibraries(libPath, processor, allowUpgrade); err != nil {
		file := filepath.Join(libPath, "libmtmd.dylib")

		if _, err := os.Stat(file); err == nil {
			return nil
		}

		return fmt.Errorf("unable to install llamacpp: %s: %w", file, err)
	}

	return nil
}

func InstallModel(modelURL string, modelPath string) (string, error) {
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
