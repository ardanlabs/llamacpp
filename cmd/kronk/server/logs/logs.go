// Package logs manages the server logs sub-command.
package logs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ardanlabs/kronk/sdk/kronk/defaults"
	"github.com/spf13/cobra"
)

func runLogs(cmd *cobra.Command, args []string) {
	logFile := logFilePath()

	tail := exec.Command("tail", "-f", logFile)
	tail.Stdout = os.Stdout
	tail.Stderr = os.Stderr

	if err := tail.Run(); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}

func logFilePath() string {
	return filepath.Join(defaults.BaseDir(""), "kronk.log")
}
