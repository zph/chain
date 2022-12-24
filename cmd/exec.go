/*
Copyright © 2022 Zander Hill <zander@xargs.io>
*/
package cmd

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec [keychain] -- [execCommand] [execCommandArgs...]",
	Short: "Execute a command in the context of ENV vars fetched from keychain",
	Long: `
	Execute a command in the context of ENV vars fetched from keychain

	eg:
	# Fetch from aws-creds keychain use aws commandline tool in that context
	chain exec aws-creds -- aws s3 ls...
`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		chain := args[0]
		command := args[1]
		commandArgs := args[1:]

		execute(cmd, chain, command, commandArgs)
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}

func execute(cmd *cobra.Command, chain string, command string, commandArgs []string) {
	lines, err := getKVAsEnvLines(cmd, chain)
	if err != nil {
		log.Fatal().Msgf("Error getting env lines: %+v", err)
	}

	env := os.Environ()
	env = append(env, lines...)

	execpath, err := exec.LookPath(command)
	if err != nil {
		log.Fatal().Msgf("Unable to find command: %s\n", command)

		cmd.Help()
		os.Exit(1)
	}

	// TODO: use golang helpers for os.exec
	err = syscall.Exec(execpath, commandArgs, env)
	log.Fatal().Msg(err.Error())
}
