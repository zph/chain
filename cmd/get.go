/*
Copyright © 2022 Zander Hill <zander@xargs.io>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get [keychain] [execCommand] [execCommandArgs...]",
	Short: "Fetch keychain values for <keychain>",
	Long: `
	Fetch keychain values for <keychain>

	eg:
	# Fetch aws-creds previously set using "chain set aws-creds"
	chain get aws-creds
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		chain := args[0]

		get(cmd, chain)
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
