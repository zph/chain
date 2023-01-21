/*
Copyright © 2022 Zander Hill <zander@xargs.io>
*/
package cmd

import (
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "chain",
	Short: "A remake of envchain and generic sibling of aws-vault",
	Long: `
chain is a tool for securely storing and retrieving secrets for use
in other cli tools.

For example:
echo "AWS_SECRET_KEY_ID=FAKEKEY" | chain set aws-creds
chain get aws-creds
chain exec aws-creds -- aws s3 ls...

# Store your one or more env variables
chain set chain-name<ENTER>

# Fetch to review them
chain get chain-name

# Execute a secondary command in the environment of these variables
chain exec chain-name -- aws s3 ...

# ENV variables
CHAIN_PASSWORD=<password used in keychain for storing key>
CHAIN_DIR=<directory for files stored on disk, default=.chain>

# Values can be set in a .chain.hcl configuration file
Use "chain init" to create the init file in .chain/.chain.hcl

Security:
- Designed to use established tools (99designs/keyring) with the File JWT backend (for portability)
- Requires min password length
- Offers to generate large secure passwords using "chain password"
- Never stores env values unencrypted on disk
`,
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var KeyringUserKey = "keyring_user"
var KeyringServiceKey = "keyring_service"
var KeyringPassword = "password"
var ChainDirKey = "dir"
var ConfigPrefix = "chain"
var PasswordValidationLength = "password_validation_length"
var KeychainBackend = "keychain_backend"
var StoreBackendTypeName = "store"
var LogLevelName = "log_level"

func init() {
	viper.SetEnvPrefix(ConfigPrefix)
	var pwd, err = os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get pwd")
	}

	viper.SetDefault(LogLevelName, "info")
	viper.SetDefault(KeyringServiceKey, ConfigPrefix)
	viper.SetDefault(KeyringUserKey, ConfigPrefix)
	viper.SetDefault(ChainDirKey, path.Join(pwd, ".", ConfigPrefix))
	viper.SetDefault(PasswordValidationLength, 20)
	viper.SetDefault(PasswordValidationLength, 20)
	viper.SetDefault(StoreBackendTypeName, 1)

	viper.BindEnv(LogLevelName)
	viper.BindEnv(KeyringServiceKey)
	viper.BindEnv(KeyringUserKey)
	viper.BindEnv(ChainDirKey)
	viper.BindEnv(KeyringPassword)
	viper.BindEnv(PasswordValidationLength)
	viper.BindEnv(StoreBackendTypeName)

	zerolog.TimestampFieldName = "t"
	zerolog.LevelFieldName = "l"
	zerolog.MessageFieldName = "m"
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	var logLevel, err2 = zerolog.ParseLevel(viper.GetString(LogLevelName))
	if err2 != nil {
		log.Fatal().Err(err).Msg("failed to parse log level")
	}

	zerolog.SetGlobalLevel(logLevel)
	zerolog.TimeFieldFormat = time.RFC3339

	localPath, err := filepath.Abs("./." + ConfigPrefix)
	if err != nil {
		log.Panic().Msgf("Unable to parse path %+v\n", localPath)
	}
	viper.SetConfigName(".chain")                                  // name of config file (without extension)
	viper.AddConfigPath("$HOME/." + viper.GetString(ConfigPrefix)) // call multiple times to add many search paths
	viper.AddConfigPath(localPath)                                 // call multiple times to add many search paths
	viper.AddConfigPath(".")                                       // optionally look for config in the working directory
	loadConfig()
}

func loadConfig() {
	// Ignore errors since the config file is optional

	_ = viper.ReadInConfig()
	//if err != nil {
	//	log.Print("error config file: %w", err)
	//}
}
