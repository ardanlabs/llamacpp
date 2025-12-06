// Package list provides the pull command code.
package list

import (
	"errors"

	"github.com/ardanlabs/kronk/defaults"
	"github.com/ardanlabs/kronk/tools"
)

var ErrInvalidArguments = errors.New("invalid arguments")

// Run executes the pull command.
func Run(args []string) error {
	modelPath := defaults.ModelsDir()

	models, err := tools.ListModels(modelPath)
	if err != nil {
		return err
	}

	tools.ListModelsFmt(models)

	return nil
}
