package list

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "list",
	Short: "List catalog models",
	Long: `List catalog models

Flags (--local mode):
      --filter-category  Filter catalogs by category name (substring match)

Environment Variables (web mode - default):
      KRONK_WEB_API_HOST  (default localhost:3000)  IP Address for the kronk server`,
	Args: cobra.ArbitraryArgs,
	Run:  runCatalogList,
}

func init() {
	Cmd.Flags().Bool("local", false, "Run without the model server")
	Cmd.Flags().String("filter-category", "", "Filter catalogs by category name (substring match)")
}

func runCatalogList(cmd *cobra.Command, args []string) {
	local, _ := cmd.Flags().GetBool("local")
	filterCategory, _ := cmd.Flags().GetString("filter-category")

	if filterCategory != "" {
		args = append(args, "--filter-category", filterCategory)
	}

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
