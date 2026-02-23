package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var appVersion = "dev"

func SetVersion(v string) {
	appVersion = v
}

var rootCmd = &cobra.Command{
	Use:   "clihub",
	Short: "Turn any MCP server into a compiled CLI binary",
	Long:  "clihub turns any MCP server into a compiled, standalone command-line tool.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	rootCmd.SetVersionTemplate(fmt.Sprintf("clihub v%s\n", appVersion))
}

func Execute() error {
	rootCmd.Version = appVersion
	return rootCmd.Execute()
}
