/*
Copyright © 2022 Zander Hill <zander@xargs.io>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

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

func get(cmd *cobra.Command, chain string) {
	lines, err := getKVAsEnvLines(cmd, chain)
	if err != nil {
		log.Fatal().Msgf("Error getting env lines: %+v", err)
	}

	fmt.Println(strings.Join(lines, "\n"))
}
