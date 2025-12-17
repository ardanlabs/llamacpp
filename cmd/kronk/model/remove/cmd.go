package remove

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "remove MODEL_NAME",
	Short: "Remove a model",
	Long: `Remove a model

Environment Variables (--local mode):
      KRONK_MODELS  (default: $HOME/kronk/models)  The path to the models directory`,
	Args: cobra.ExactArgs(1),
	Run:  runRemove,
}

func init() {
	Cmd.Flags().Bool("local", false, "Run without the model server")
}

func runRemove(cmd *cobra.Command, args []string) {
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
