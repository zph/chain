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
		commandArgs := args[2:]

		err := execute(cmd, chain, command, commandArgs)
		if err != nil {
			log.Fatal().Err(err).Str("command", command).Msgf("failed to run syscall %+v with args: %+v", command, commandArgs)
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(execCmd)
}

func execute(cmd *cobra.Command, chain string, command string, commandArgs []string) error {
	lines, err := getKVAsEnvLines(cmd, chain)
	if err != nil {
		log.Fatal().Msgf("Error getting env lines: %+v", err)
	}

	env := os.Environ()
	env = append(env, lines...)

	argv0, err := exec.LookPath(command)
	if err != nil {
		log.Fatal().Msgf("Unable to find command: %s\n", command)

		cmd.Help()
		os.Exit(1)
	}

	// Credit: https://raw.githubusercontent.com/99designs/aws-vault/master/cli/exec.go
	argv := make([]string, 0, 1+len(commandArgs))
	argv = append(argv, argv0)
	argv = append(argv, commandArgs...)

	log.Debug().Str("command", command).Strs("args", argv).Strs("env", env).Msg("executing syscall")
	// TODO: use golang helpers for os.exec
	return syscall.Exec(argv0, argv, env)
}
