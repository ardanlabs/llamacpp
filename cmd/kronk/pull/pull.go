// Package pull provides the pull command code.
package pull

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/ardanlabs/kronk/cmd/kronk/defaults"
	"github.com/ardanlabs/kronk/install"
)

var ErrInvalidArguments = errors.New("invalid arguments")

// Run executes the pull command.
func Run(args []string) error {
	if len(args) == 0 {
		return ErrInvalidArguments
	}

	modelPath := defaults.ModelsDir()
	modelURL := args[0]

	if _, err := url.ParseRequestURI(modelURL); err != nil {
		return fmt.Errorf("invalid URL: %s", modelURL)
	}

	fmt.Println("ModelURL :", modelURL)
	fmt.Println("ModelPath:", modelPath)

	f := func(src string, currentSize int64, totalSize int64, mibPerSec float64, complete bool) {
		fmt.Printf("\r\x1b[KDownloading %s... %d MiB of %d MiB (%.2f MiB/s)", src, currentSize/(1024*1024), totalSize/(1024*1024), mibPerSec)
		if complete {
			fmt.Println()
		}
	}

	_, downloaded, err := install.ModelWithProgress(modelURL, modelPath, f)
	if err != nil {
		return err
	}

	if !downloaded {
		fmt.Println("Already downloaded")
		return nil
	}

	fmt.Println("Download Completed")
	return nil
}
