package cmd

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/99designs/keyring"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/proto"

	chainv1 "github.com/zph/chain/gen/go/chain/v1"
)

type Store interface {
	Keys() ([]string, error)
	Get(string) (keyring.Item, error)
	Set(keyring.Item) error
	Name() string
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
	}
	return nil, eris.New("Store type unfound, choose from chainv1.StorageType enum")
}

type KeychainByPlatformStore struct {
	keyring.Keyring
}

func NewKeychainByPlatform(chain string) (Store, error) {
	s := KeychainByPlatformStore{}
	namespacedChain := fmt.Sprintf("%s:%s", ConfigPrefix, chain)
	// Available backends are used in descending order by Operating system
	// which requires supplying config for many variants
	// See: https://github.com/99designs/keyring/blob/master/keyring.go#L27-L39
	k, err := keyring.Open(keyring.Config{
		AllowedBackends:          keyring.AvailableBackends(),
		ServiceName:              chain,
		KeychainName:             namespacedChain,
		KeychainTrustApplication: true,
		FilePasswordFunc:         getPassword,
		FileDir:                  filePath(chain),
		// KeyCtlScope is the scope of the kernel keyring (either "user", "session", "process" or "thread")
		KeyCtlScope: "session",

		// KeyCtlPerm is the permission mask to use for new keys
		// KeyCtlPerm uint32

	})
	if err != nil {
		return nil, err
	}

	s.Keyring = k
	return s, nil
}

func (s KeychainByPlatformStore) Name() string {
	return chainv1.StorageType_STORAGE_TYPE_KEYCHAIN_BY_PLATFORM.String()
}

// TODO: consider supporting other backends:
// https://pkg.go.dev/github.com/99designs/keyring#BackendType
func NewStandardStore(chain string) (Store, error) {
	s := StandardStore{}
	k, err := keyring.Open(keyring.Config{
		AllowedBackends:  []keyring.BackendType{keyring.FileBackend},
		ServiceName:      chain,
		FilePasswordFunc: getPassword,
		FileDir:          filePath(chain),
	})
	if err != nil {
		return nil, err
	}

	s.Keyring = k
	return s, nil
}

type StandardStore struct {
	keyring.Keyring
}

func (s StandardStore) Name() string {
	return chainv1.StorageType_STORAGE_TYPE_STANDARD_STORE.String()
}

type MetadataEncodedStore struct {
	k keyring.Keyring
}

func (s MetadataEncodedStore) Name() string {
	return chainv1.StorageType_STORAGE_TYPE_METADATA_ENCODED_STORE.String()
}

var ErrFunctionNotImplemented = errors.New("function not implemented")

var MetaDataName = "METADATA"

func NewMetadataEncodedStore(chain string) (Store, error) {
	return nil, eris.Wrapf(ErrFunctionNotImplemented, "NewMetadataEncodedStore for %+v", chain)

	s := MetadataEncodedStore{}
	k, err := keyring.Open(keyring.Config{
		AllowedBackends:  []keyring.BackendType{keyring.FileBackend},
		ServiceName:      chain,
		FilePasswordFunc: getPassword,
		FileDir:          filePath(chain),
	})
	if err != nil {
		return nil, err
	}

	s.k = k
	return s, nil
}

func (s MetadataEncodedStore) ensureData() error {
	var meta chainv1.Storage
	item, err := s.k.Get(MetaDataName)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		idx := make(map[string]*chainv1.IndexEntry)
		meta = chainv1.Storage{Type: chainv1.StorageType_STORAGE_TYPE_METADATA_ENCODED_STORE, ReverseIndex: idx}
		log.Debug().Msgf("Metadata after init: %+v", meta)
	} else {
		proto.Unmarshal(item.Data, &meta)
	}
	b, err := proto.Marshal(&meta)
	if err != nil {
		log.Fatal().Msgf("Unable to marshall proto in ensureData: %+v", err)
	}
	s.k.Set(keyring.Item{Key: MetaDataName, Data: b})
	return nil
}

func (s MetadataEncodedStore) GetMeta() (chainv1.Storage, error) {
	var meta chainv1.Storage
	item, err := s.k.Get(MetaDataName)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return chainv1.Storage{}, err
	}
	proto.Unmarshal(item.Data, &meta)

	return meta, nil
}

// Set
func (s MetadataEncodedStore) Set(item keyring.Item) error {
	s.ensureData()
	// write to record then to reverse index
	// Store value as proto of k/v for recreating these
	meta, err := s.GetMeta()
	if err != nil {
		log.Fatal().Msgf("Unable to get reverse index %+v", err)
	}

	uuid := uuid.New()
	itemWithUUID := item
	itemWithUUID.Key = uuid.String()
	log.Printf("METADATA: %+v", meta)

	meta.ReverseIndex[item.Key] = &chainv1.IndexEntry{Key: uuid.String(), Value: item.Data}
	b, err := proto.Marshal(&meta)
	if err != nil {
		log.Fatal().Msgf("Unable to marshall reverse index %+v", err)
	}

	s.k.Set(itemWithUUID)
	s.k.Set(keyring.Item{Key: MetaDataName, Data: b})
	return nil
}

// Get
// Read from reverse index, then get record
// Store value as proto of k/v for recreating these
func (s MetadataEncodedStore) Get(envKey string) (keyring.Item, error) {
	s.ensureData()
	meta, err := s.GetMeta()
	if err != nil {
		log.Fatal().Msgf("Unable to get reverse index %+v", err)
	}

	reverseIndexItem := meta.ReverseIndex[envKey]
	uuidKey := reverseIndexItem.Key

	i, err := s.k.Get(uuidKey)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		log.Fatal().Msgf("Unable to fetch key %+v %+v", err, i)
	}

	return i, nil
}

// TODO
// List all keys, then perform translation in reverse index
func (s MetadataEncodedStore) Keys() ([]string, error) {
	s.ensureData()
	return nil, ErrFunctionNotImplemented
}
