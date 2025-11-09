package cmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/99designs/keyring"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"

	chainv1 "github.com/zph/chain/gen/go/chain/v1"
)

var secureFSPerm fs.FileMode = 0600
var ErrFunctionNotImplemented = errors.New("function not implemented")

type Store interface {
	Keys() ([]string, error)
	Get(string) (keyring.Item, error)
	Set(keyring.Item) error
	Remove(string) error
	Name() string
	PostRunHook() error
}

// saveStoreTypeToConfig writes the store type to the HCL config file
func saveStoreTypeToConfig(storeType int32) error {
	viper.Set(StoreBackendTypeName, storeType)

	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		// If no config file exists, create one in the default location
		chainDir := viper.GetString(ChainDirKey)
		configFile = filepath.Join(chainDir, ".chain.hcl")
		viper.SetConfigFile(configFile)

		// Ensure directory exists
		if err := os.MkdirAll(chainDir, 0700); err != nil {
			return eris.Wrap(err, "failed to create config directory")
		}
	}

	if err := viper.WriteConfig(); err != nil {
		// If config doesn't exist, use SafeWriteConfig
		if os.IsNotExist(err) {
			if err := viper.SafeWriteConfig(); err != nil {
				return eris.Wrap(err, "failed to write config file")
			}
		} else {
			return eris.Wrap(err, "failed to save config")
		}
	}

	log.Debug().Int32("store_type", storeType).Str("config_file", configFile).Msg("Saved store type to config")
	return nil
}

func NewStore(chain string) (Store, error) {
	// Get store type from config/env/CLI
	storeType := viper.GetInt32(StoreBackendTypeName)

	// If store type is explicitly set via env var or CLI flag (not just the default),
	// save it to the config file for future use
	if storeType != 0 && viper.IsSet(StoreBackendTypeName) {
		if err := saveStoreTypeToConfig(storeType); err != nil {
			log.Warn().Err(err).Msg("Failed to save store type to config")
		}
	}

	name := chainv1.StorageType_name[storeType]
	log.Debug().Int32("store_type", storeType).Str("store_options", name).Msg("")
	switch name {
	case chainv1.StorageType_STORAGE_TYPE_METADATA_ENCODED_STORE.String():
		return NewMetadataEncodedStore(chain)
	case chainv1.StorageType_STORAGE_TYPE_STANDARD_STORE.String():
		return NewStandardStore(chain)
	case chainv1.StorageType_STORAGE_TYPE_KEYCHAIN_BY_PLATFORM.String():
		return NewKeychainByPlatform(chain)
	case chainv1.StorageType_STORAGE_TYPE_AGE_STORE.String():
		return NewAgeStore(chain)
	case chainv1.StorageType_STORAGE_TYPE_AGE_OTP_STORE.String():
		return NewAgeOTPStore(chain)
	case chainv1.StorageType_STORAGE_TYPE_SQLITE.String():
		return NewSQLiteStore(chain)
	}
	return nil, eris.New("Store type unfound, choose from chainv1.StorageType enum")
}
