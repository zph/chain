package cmd

import (
	"fmt"

	"github.com/99designs/keyring"
	chainv1 "github.com/zph/chain/gen/go/chain/v1"
)

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

func (s KeychainByPlatformStore) PostRunHook() error { return nil }

func (s KeychainByPlatformStore) Name() string {
	return chainv1.StorageType_STORAGE_TYPE_KEYCHAIN_BY_PLATFORM.String()
}
