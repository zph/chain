/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// createKeysCmd represents the createKeys command
var createKeysCmd = &cobra.Command{
	Use:   "createKeys",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		chain := args[0]
		amount := 10
		outputPath := path.Join(filePath(chain), publicKeyFile)

		err := os.MkdirAll(filePath(chain), 0700)
		if err != nil && !os.IsExist(err) {
			log.Fatal().Err(err).Msg("")
		}
		ids := createIdentities(amount)

		err = setPublicKeys(ids, outputPath)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		privateKeys := ""
		for _, id := range ids {
			privateKeys += id.Recipient().String()[:10] + ":" + id.String() + "\n"
		}

		fmt.Printf("# Store these for one time use\n%s", privateKeys)
	},
}

func init() {
	rootCmd.AddCommand(createKeysCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createKeysCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createKeysCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
