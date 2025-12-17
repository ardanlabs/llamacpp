package create

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "create",
	Short: "Create a security token",
	Long: `Create a security token

Flags:
      --admin-token  Admin token for authentication
      --duration     Token duration (e.g., 1h, 1d, 1m, 1y)
      --endpoints    Comma-separated list of allowed endpoints`,
	Args: cobra.NoArgs,
	Run:  runTokenCreate,
}

func init() {
	Cmd.Flags().Bool("local", false, "Run without the model server")
	Cmd.Flags().String("duration", "", "Token duration (e.g., 1h, 1d, 1m, 1y)")
	Cmd.Flags().StringSlice("endpoints", []string{}, "Comma-separated list of allowed endpoints")
}

func runTokenCreate(cmd *cobra.Command, args []string) {
	local, _ := cmd.Flags().GetBool("local")
	adminToken, _ := cmd.Flags().GetString("admin-token")
	duration, _ := cmd.Flags().GetString("duration")
	endpoints, _ := cmd.Flags().GetStringSlice("endpoints")

	cfg := Config{
		AdminToken: adminToken,
		Duration:   duration,
		Endpoints:  endpoints,
	}

	var err error

	switch local {
	case true:
		err = runLocal(cfg)
	default:
		err = runWeb(cfg)
	}

	if err != nil {
		fmt.Println("\nERROR:", err)
		os.Exit(1)
	}
}
