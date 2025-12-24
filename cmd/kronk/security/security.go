// Package security provides tooling support for security.
package security

import (
	"errors"
	"os"

	"github.com/ardanlabs/kronk/cmd/kronk/security/key"
	"github.com/ardanlabs/kronk/cmd/kronk/security/sec"
	"github.com/ardanlabs/kronk/cmd/kronk/security/token"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "security",
	Short: "Manage security",
	Long: `Manage security - tokens and access control

Environment Variables:
  KRONK_TOKEN    Admin level token required for authentication. Must be set
                 before running any security commands.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("KRONK_TOKEN") == "" {
			return errors.New("KRONK_TOKEN environment variable must be set")
		}
		return sec.Authenticate()
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(key.Cmd)
	Cmd.AddCommand(token.Cmd)
}
