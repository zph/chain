package cmd

import (
	"github.com/99designs/keyring"
	chainv1 "github.com/zph/chain/gen/go/chain/v1"
)

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
func (s StandardStore) PostRunHook() error { return nil }
