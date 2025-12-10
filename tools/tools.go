// Package tools provides functions for installing and upgrading llama.cpp.
package tools

import (
	"os"
)

func fileSize(filePath string) (int, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}

	return int(info.Size()), nil
}
