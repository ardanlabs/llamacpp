package libs

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "libs",
	Short: "Install or upgrade llama.cpp libraries",
	Long: `Install or upgrade llama.cpp libraries

Environment Variables (web mode - default):
      KRONK_WEB_API_HOST  (default localhost:3000)  IP Address for the kronk server.

Environment Variables (--local mode):
      KRONK_ARCH       (default: runtime.GOARCH)         The architecture to install.
      KRONK_LIB_PATH   (default: $HOME/kronk/libraries)  The path to the libraries directory,
      KRONK_OS         (default: runtime.GOOS)           The operating system to install.
      KRONK_PROCESSOR  (default: cpu)                    Options: cpu, cuda, metal, vulkan`,
	Args: cobra.NoArgs,
	Run:  runLibs,
}

func init() {
	Cmd.Flags().Bool("local", false, "Run without the model server")
}

func runLibs(cmd *cobra.Command, args []string) {
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
