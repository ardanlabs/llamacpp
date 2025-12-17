package pull

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "pull <MODEL_ID>",
	Short: "Pull a model from the catalog",
	Long: `Pull a model from the catalog

Environment Variables (web mode - default):
      KRONK_WEB_API_HOST  (default localhost:3000)  IP Address for the kronk server`,
	Args: cobra.ExactArgs(1),
	Run:  runCatalogPull,
}

func init() {
	Cmd.Flags().Bool("local", false, "Run without the model server")
}

func runCatalogPull(cmd *cobra.Command, args []string) {
	local, _ := cmd.Flags().GetBool("local")

	var err error

	switch local {
	case true:
		err = runLocal(args)
	default:
		err = runWeb(args)
	}

	if err != nil {
		fmt.Println("\nERROR:", err)
		os.Exit(1)
	}
}
