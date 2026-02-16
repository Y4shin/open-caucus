package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "conference-tool",
	Short: "Conference management tool",
}

func Execute() error {
	return rootCmd.Execute()
}
