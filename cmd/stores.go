package cmd

import (
	"errors"
	"io/fs"

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

func NewStore(chain string) (Store, error) {
	storeType := viper.GetInt32(StoreBackendTypeName)
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
	}
	return nil, eris.New("Store type unfound, choose from chainv1.StorageType enum")
}
