package start

import (
	"fmt"
	"os"

	"github.com/ardanlabs/kronk/cmd/server/api/services/kronk"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "start",
	Short: "Start Kronk model server",
	Long:  `Start Kronk model server. Use --help to get environment settings`,
	Args:  cobra.NoArgs,
	Run:   main,
}

func init() {
	Cmd.Flags().BoolP("detach", "d", false, "Run server in the background")

	Cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		err := kronk.Run(true)
		cmd.Long = fmt.Sprintf("Start Kronk model server\n\n%s", err.Error())
		cmd.Parent().HelpFunc()(cmd, args)
	})
}

func main(cmd *cobra.Command, args []string) {
	if err := run(cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command) error {
	return runLocal(cmd)
}
