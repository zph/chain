package cmd

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"filippo.io/age"
	"github.com/99designs/keyring"
	"github.com/rs/zerolog/log"
	chainv1 "github.com/zph/chain/gen/go/chain/v1"
)

var publicKeyFile = ".PUBLIC_KEYS"

// AgeStore is used for both AgeStore and AgeOTPStore
func NewAgeStore(chain string) (Store, error) {
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

func (s AgeStore) getKeysFromUser() (string, string, error) {
	pk, err := s.Config.FilePasswordFunc("")
	if err != nil {
		return "", "", err
	}

	publicKeyPrefix, privateKey, found := strings.Cut(pk, ":")
	if !found {
		return "", "", err
	}

	return publicKeyPrefix, privateKey, nil
}

func (s AgeStore) Get(key string) (keyring.Item, error) {
	_, privateKey, err := s.getKeysFromUser()
	if err != nil {
		return keyring.Item{}, err
	}

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

	err = os.WriteFile(s.FilePath(item.Key), out.Bytes(), secureFSPerm)
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
	recipients, err := age.ParseRecipients(publicKeys)
	if err != nil {
		log.Fatal().Msgf("Failed to parse public keys: %v", err)
	}
	return recipients
}

func setPublicKeys(recipients []string, outputPath string) error {
	log.Debug().Str("path", outputPath).Msgf("recipients %+v", recipients)
	err := os.WriteFile(outputPath, []byte(strings.Join(recipients, "\n")), secureFSPerm)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return nil
}

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
	for range count {
		identity, err := age.GenerateX25519Identity()
		if err != nil {
			log.Fatal().Msgf("Failed to generate key pair: %v", err)
		}

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

func (s AgeStore) PostRunHook() error { return nil }
