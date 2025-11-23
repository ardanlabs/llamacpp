package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ardanlabs/kronk"
	"github.com/hybridgroup/yzma/pkg/download"
)

const (
	libPath = "libraries"
)

func main() {
	if err := run(); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}

func run() error {
	vi, err := kronk.RetrieveVersionInfo(libPath)
	if err != nil {
		return fmt.Errorf("error retrieving version info: %w", err)
	}

	fmt.Println()

	if vi.Current == vi.Latest {
		fmt.Println("Llamacpp is up to date")
		fmt.Printf("Latest version: %s\nCurrent version: %s\n", vi.Latest, vi.Current)
		return nil
	}

	fmt.Println("Installing Llamacpp")

	vi, err = kronk.InstallLlama(libPath, download.CPU, true)
	if err != nil {
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

	fmt.Printf("Latest version: %s\nCurrent version: %s\n", vi.Latest, vi.Current)

	return nil
}
