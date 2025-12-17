package logs

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream server logs",
	Long:  `Stream the Kronk model server logs (tail -f)`,
	Args:  cobra.NoArgs,
	Run:   runLogs,
}
