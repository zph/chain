package cmd

import (
	"bytes"
	"os"

	"github.com/99designs/keyring"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	chainv1 "github.com/zph/chain/gen/go/chain/v1"
)

// TODO: consider adding re-keying option for the scenario
// where user has 1 valid private key remaining and would
// like to keep using the same stored secrets with N new
// keys.

// AgeStore is used for both AgeStore and AgeOTPStore
func NewAgeOTPStore(chain string) (Store, error) {
	s := AgeOTPStore{}
	cfg := keyring.Config{
		ServiceName:      chain,
		FilePasswordFunc: getPassword,
		FileDir:          filePath(chain),
	}
	s.Config = cfg
	return s, nil
}

type AgeOTPStore struct {
	AgeStore
}

func (s AgeOTPStore) Name() string {
	return chainv1.StorageType_STORAGE_TYPE_AGE_OTP_STORE.String()
}

func (s AgeOTPStore) PostRunHook() error {
	// Fetch the publicKeyPrefix:PrivateKey string from user
	publicKeyPrefix, _, err := s.getKeysFromUser()
	if err != nil {
		return err
	}

	// Remove public key from list of keys
	err = s.expirePublicKey(publicKeyPrefix)
	if err != nil {
		return eris.Wrap(err, "failed to expire keys and rekey")
	}

	// Rekey existing records after now that we've expired
	// the one-time-use public/private keypair
	var keys []string
	keys, err = s.Keys()
	if err != nil {
		return eris.Wrap(err, "failed to list keys")
	}
	for _, k := range keys {
		var item keyring.Item
		item, err = s.Get(k)
		if err != nil {
			return eris.Wrapf(err, "failed to fetch key: %+v", k)
		}
		err = s.Set(item)
		if err != nil {
			return eris.Wrapf(err, "failed to set key using new public keys: %+v", k)
		}
	}
	return nil
}

func (s AgeOTPStore) expirePublicKey(publicKeyPrefix string) error {
	content, err := os.ReadFile(s.FilePath(publicKeyFile))
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to open public key file")
	}

	lines := bytes.Split(content, []byte("\n"))
	var newLines [][]byte
	for _, l := range lines {
		if string(l)[:10] != publicKeyPrefix {
			newLines = append(newLines, l)
		}
	}

	os.WriteFile(s.FilePath(publicKeyFile), bytes.Join(newLines, []byte("\n")), secureFSPerm)
	return nil
}
