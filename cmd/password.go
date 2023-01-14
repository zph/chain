/*
Copyright © 2022 Zander Hill <zander@xargs.io>
*/
package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/sethvargo/go-password/password"
	"github.com/spf13/cobra"
)

// passwordCmd represents the password command
var passwordCmd = &cobra.Command{
	Use:   "password",
	Short: "Generates secure password",
	Long: `
	Generates a secure password of 64 chars with ints and symbols
`,
	Run: DoCreatePassword,
}

func DoCreatePassword(cmd *cobra.Command, args []string) {
	// Disallow symbols that might have special annoying meaning in shell
	gen, _ := password.NewGenerator(&password.GeneratorInput{
		Symbols: "!_+=:?,.^",
	})

	// Generate a password that is 64 characters long with 10 digits, 10 symbols,
	// allowing upper and lower case letters, allowing repeat characters.
	res, err := gen.Generate(64, 10, 10, false, true)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	fmt.Println(res)
}

func init() {
	RootCmd.AddCommand(passwordCmd)
}
