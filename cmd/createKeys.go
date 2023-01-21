/*
Copyright © 2022 Zander Hill <zander@xargs.io>
*/
package cmd

import (
	"errors"
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
	Use:   "create-keys keychain keyCount",
	Short: "Create keys which will be used with AGE backends",
	Long: `
	chain create-keys [keychain] [keyCount]
`,
	Args: cobra.ExactArgs(2),
	Run:  DoCreateKeyCmd,
}

func DoCreateKeyCmd(cmd *cobra.Command, args []string) {
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
	if _, err := os.Stat(outputPath); errors.Is(err, os.ErrNotExist) {
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

		fmt.Printf("# Store these keys for decryption.\nIf using age-otp-store, each one will be expired upon use.\n%s\n", privateKeys)
	} else {
		log.Fatal().Str("outputPath", outputPath).Msg("Exiting because .PUBLIC_KEY already exists. Delete and re-run")
	}
}

func init() {
	RootCmd.AddCommand(createKeysCmd)
}
