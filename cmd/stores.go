package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"filippo.io/age"
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
	Remove(string) error
	Name() string
}

func NewAgeStore(chain string) (Store, error) {
	log.Warn().Msgf("EXPERIMENTAL STORE")

	s := AgeStore{}
	cfg := keyring.Config{
		ServiceName:      chain,
		FilePasswordFunc: getPassword,
		FileDir:          filePath(chain),
	}
	s.Config = cfg
	return s, nil
}

type AgeStore struct {
	Config keyring.Config
	// https://github.com/FiloSottile/age
}

func (s AgeStore) Keys() ([]string, error) {
	files, err := ioutil.ReadDir(s.Config.FileDir)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	var output []string
	for _, f := range files {
		if f.Name() != publicKeyFile {
			output = append(output, f.Name())
		}
	}
	return output, nil
}

func (s AgeStore) Get(key string) (keyring.Item, error) {
	pk, err := s.Config.FilePasswordFunc("")
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	publicKeyPrefix, privateKey, found := strings.Cut(pk, ":")
	if !found {
		log.Fatal().Msg("")
	}
	recipients := s.getRecipients()
	var otherRecipients []age.Recipient
	for _, r := range recipients {
		otherRecipients = append(otherRecipients, r)
	}
	log.Debug().Str("publicKeyPrefix", publicKeyPrefix).Msg("")

	identity, err := age.ParseX25519Identity(strings.TrimSpace(privateKey))
	if err != nil {
		log.Fatal().Msgf("Failed to parse private key: %v", err)
	}

	credsFile := s.FilePath(key)
	f, err := os.Open(credsFile)
	if err != nil {
		log.Fatal().Str("credsFile", credsFile).Err(err).Msg("Failed to open file")
	}

	log.Debug().Str("credsFile", credsFile).Msg("Opened file")
	r, err := age.Decrypt(f, identity)
	if err != nil {
		log.Fatal().Msgf("Failed to open encrypted file: %v", err)
	}
	out := &bytes.Buffer{}
	if _, err := io.Copy(out, r); err != nil {
		log.Fatal().Msgf("Failed to read encrypted file: %v", err)
	}

	fmt.Printf("File contents: %q\n", out.Bytes())
	return keyring.Item{Key: key, Data: out.Bytes()}, nil
}

// Load Public Keys
// Using pub keys, encode item
func (s AgeStore) Set(item keyring.Item) error {
	// https://pkg.go.dev/filippo.io/age#example-Encrypt

	recipients := s.getRecipients()

	out := &bytes.Buffer{}

	w, err := age.Encrypt(out, recipients...)
	if err != nil {
		log.Fatal().Msgf("Failed to create encrypted file: %v", err)
	}
	if _, err := io.WriteString(w, string(item.Data)); err != nil {
		log.Fatal().Msgf("Failed to write to encrypted file: %v", err)
	}
	if err := w.Close(); err != nil {
		log.Fatal().Msgf("Failed to close encrypted file: %v", err)
	}

	err = os.WriteFile(s.FilePath(item.Key), out.Bytes(), 0600)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return nil
}

func (s AgeStore) getRecipients() []age.Recipient {
	publicKeys, err := os.Open(s.FilePath(publicKeyFile))

	if err != nil {
		log.Fatal().Msgf("Failed to open private keys file: %v", err)
	}
	// TODO continue here on XMAS
	recipients, err := age.ParseRecipients(publicKeys)
	if err != nil {
		log.Fatal().Msgf("Failed to parse public keys: %v", err)
	}
	return recipients
}

func setPublicKeys(identities []*age.X25519Identity, outputPath string) error {
	var recipients []string
	for _, r := range identities {
		recipients = append(recipients, r.Recipient().String())
	}
	log.Debug().Msgf("recipients %+v", recipients)
	log.Debug().Str("path", outputPath).Msgf("recipients %+v", recipients)
	err := os.WriteFile(outputPath, []byte(strings.Join(recipients, "\n")), 0600)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return nil
}

var publicKeyFile = ".PUBLIC_KEYS"

func (s AgeStore) publicKeysExist() (bool, error) {
	if _, err := os.Stat(s.FilePath(publicKeyFile)); err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, err
	}
}

func createIdentities(amt int) []*age.X25519Identity {
	count := make([]string, amt)

	var identities []*age.X25519Identity
	for _ = range count {
		identity, err := age.GenerateX25519Identity()
		if err != nil {
			log.Fatal().Msgf("Failed to generate key pair: %v", err)
		}

		log.Debug().Msgf("Public key: %s...\n", identity.Recipient().String())
		log.Debug().Msgf("Private key: %s...\n", identity.String())

		identities = append(identities, identity)
	}
	return identities
}

func (s AgeStore) Remove(key string) error {
	return ErrFunctionNotImplemented
}

func (s AgeStore) FilePath(key string) string {
	return path.Join(s.Config.FileDir, key)
}

func (s AgeStore) Name() string {
	return chainv1.StorageType_STORAGE_TYPE_AGE_STORE.String()
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

func (s MetadataEncodedStore) Remove(key string) error {
	return ErrFunctionNotImplemented
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
