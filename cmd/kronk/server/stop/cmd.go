package stop

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Kronk model server",
	Long:  `Stop the Kronk model server by sending SIGTERM`,
	Args:  cobra.NoArgs,
	Run:   runStop,
}

func runStop(cmd *cobra.Command, args []string) {
	if err := runLocal(); err != nil {
		fmt.Println("\nERROR:", err)
		os.Exit(1)
	}
}
