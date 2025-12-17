package list

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "list",
	Short: "List models",
	Long: `List models

Environment Variables (web mode - default):
      KRONK_WEB_API_HOST  (default localhost:3000)  IP Address for the kronk server

Environment Variables (--local mode):
      KRONK_MODELS  (default: $HOME/kronk/models)  The path to the models directory`,
	Args: cobra.NoArgs,
	Run:  runList,
}

func init() {
	Cmd.Flags().Bool("local", false, "Run without the model server")
}

func runList(cmd *cobra.Command, args []string) {
	local, _ := cmd.Flags().GetBool("local")

	var err error

	switch local {
	case true:
		err = runLocal()
	default:
		err = runWeb()
	}

	if err != nil {
		fmt.Println("\nERROR:", err)
		os.Exit(1)
	}
}
