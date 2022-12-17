/*
	  Copyright ©
	  2022-     Zander Hill <zander@xargs.io>
		0000-2020 Evan Phoenix <evan@phx.io>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "chain",
	Short: "A brief description of your application",
	Long: `


# ENV variables
CHAIN_KEYRING_SERVICE=<service/namespace used in keychain for storing key, default=schain>
CHAIN_KEYRING_USER=<username used in keychain for storing key, use for different roles, default=schain>
CHAIN_DIR=<directory for files stored on disk, default=.schain>
`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var KeyringUserKey = "keyring_user"
var KeyringServiceKey = "keyring_service"
var ChainDirKey = "dir"
var ConfigPrefix = "chain"

func init() {
	viper.SetDefault(KeyringServiceKey, ConfigPrefix)
	viper.SetDefault(KeyringUserKey, ConfigPrefix)
	viper.SetDefault(ChainDirKey, "."+ConfigPrefix)

	viper.SetEnvPrefix(ConfigPrefix)
	viper.BindEnv(KeyringServiceKey)
	viper.BindEnv(KeyringUserKey)
	viper.BindEnv(ChainDirKey)
	// TODO: add verbose and logging mode controls
}
