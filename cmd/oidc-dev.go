package cmd

import "github.com/spf13/cobra"

var oidcDevCmd = &cobra.Command{
	Use:   "oidc-dev",
	Short: "Local OIDC development utilities",
}

func init() {
	rootCmd.AddCommand(oidcDevCmd)
}
