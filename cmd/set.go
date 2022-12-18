/*
Copyright © 2022 Zander Hill <zander@xargs.io>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set [keychain]",
	Short: "Set a key in keychain",
	Long: `chain set:
	Set a value in the keychain

	Example:
	$ chain set keychain-name<ENTER>
	$> EXAMPLE_KEY="example-value"
	$> <enter on empty line to save>

	Via pipeline
	$ echo EXAMPLE_KEY="example-value" | chain set keychain-name
	`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		chain := args[0]

		set(cmd, chain)
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
}
