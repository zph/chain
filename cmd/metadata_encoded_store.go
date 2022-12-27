package cmd

import (
	"errors"

	"github.com/99designs/keyring"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	chainv1 "github.com/zph/chain/gen/go/chain/v1"
)

type MetadataEncodedStore struct {
	k keyring.Keyring
}

func (s MetadataEncodedStore) PostRunHook() error { return nil }

func (s MetadataEncodedStore) Name() string {
	return chainv1.StorageType_STORAGE_TYPE_METADATA_ENCODED_STORE.String()
}

func (s MetadataEncodedStore) Remove(key string) error {
	return ErrFunctionNotImplemented
}

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
