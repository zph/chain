/*
Copyright © 2022 Zander Hill <zander@xargs.io>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	chainv1 "github.com/zph/chain/gen/go/chain/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	Args:    cobra.ExactArgs(1),
	Run:     DoGet,
	PostRun: DoGetPostRun,
}

func init() {
	RootCmd.AddCommand(getCmd)
}

func DoGet(cmd *cobra.Command, args []string) {
	chain := args[0]

	lines, err := getKVAsEnvLines(cmd, chain)
	if err != nil {
		log.Fatal().Msgf("Error getting env lines: %+v", err)
	}

	fmt.Println(strings.Join(lines, "\n"))
}

func DoGetPostRun(cmd *cobra.Command, args []string) {
	chain := args[0]
	store, err := NewStore(chain)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if viper.GetInt32(StoreBackendTypeName) == int32(chainv1.StorageType_STORAGE_TYPE_AGE_OTP_STORE) {
		err = store.PostRunHook()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed in getPostRun for AgeOTP")
		}
	}
}
