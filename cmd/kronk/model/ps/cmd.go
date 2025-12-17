package ps

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "ps",
	Short: "List running models",
	Long: `List running models

Environment Variables:
      KRONK_WEB_API_HOST  (default localhost:3000)  IP Address for the kronk server`,
	Run: runPs,
}

func runPs(cmd *cobra.Command, args []string) {
	if err := runWeb(); err != nil {
		fmt.Println("\nERROR:", err)
		os.Exit(1)
	}
}
