/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	chainv1 "github.com/zph/chain/gen/go/chain/v1"
)

// createKeysCmd represents the createKeys command
var createKeysCmd = &cobra.Command{
	Use:   "create-keys",
	Short: "Create keys which will be used with AGE backends",
	Long: `

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if !(viper.GetInt(StoreBackendTypeName) == int(chainv1.StorageType_STORAGE_TYPE_AGE_STORE) ||
			viper.GetInt(StoreBackendTypeName) == int(chainv1.StorageType_STORAGE_TYPE_AGE_OTP_STORE)) {
			log.Fatal().Msg("create-keys only supported for AGE backends")
			os.Exit(2)
		}

		chain := args[0]
		amount, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatal().Err(err).Msg("failed to parse input for amount")
		}
		outputPath := path.Join(filePath(chain), publicKeyFile)

		err = os.MkdirAll(filePath(chain), 0700)
		if err != nil && !os.IsExist(err) {
			log.Fatal().Err(err).Msg("")
		}
		ids := createIdentities(amount)

		var recipients []string
		for _, r := range ids {
			recipients = append(recipients, r.Recipient().String())
		}

		err = setPublicKeys(recipients, outputPath)
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
