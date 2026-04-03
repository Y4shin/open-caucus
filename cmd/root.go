package cmd

import "github.com/spf13/cobra"

// version is set at build time via -ldflags "-X github.com/Y4shin/open-caucus/cmd.version=x.y.z"
var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "conference-tool",
	Short:   "Conference management tool",
	Version: version,
}

func Execute() error {
	return rootCmd.Execute()
}
