package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "1.0.1"

var versioncmd = &cobra.Command{
	Use:   "version",
	Short: "Prints version of the app",
	Long:  "Prints version of the app",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func init() {
	rootCmd.AddCommand(versioncmd)
}
