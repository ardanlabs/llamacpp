// Package security provides tooling support for security.
package security

import (
	"github.com/ardanlabs/kronk/cmd/kronk/security/token"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "security",
	Short: "Manage security",
	Long:  `Manage security - tokens and access control`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(token.Cmd)
}
