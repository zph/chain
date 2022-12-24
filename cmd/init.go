/*
Copyright © 2022 Zander Hill <zander@xargs.io>
*/
package cmd

import (
	"os"

	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create config file for chain",
	Long: `
	Creates config file for chain with any currently set values.

	chain init
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := os.Mkdir("."+ConfigPrefix, 0700)
		if err != nil && !os.IsExist(err) {
			log.Fatal().Msg(err.Error())
		}
		configFile := "." + ConfigPrefix + "/.chain.hcl"
		// Setting one empty, which will be rejected as invalid for safety
		viper.Set(KeyringPassword, "")

		// Removing these because they're currently not used
		viper.Set(KeyringServiceKey, "")
		viper.Set(KeyringUserKey, "")
		err = viper.SafeWriteConfigAs(configFile)
		if err != nil {
			log.Fatal().Msgf("Unable write file %+v\n", err)
		}
		err = os.Chmod(configFile, 0600)
		if err != nil {
			log.Fatal().Msgf("Unable to set permissions on file %s error: %+v\n", configFile, err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
