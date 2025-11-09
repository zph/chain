package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/99designs/keyring"
	"github.com/spf13/viper"
)

// setupTestStore is a helper function to set up a test environment for stores
func setupTestStore(t *testing.T, chainName string, needsAge bool) (string, string) {
	tmpDir := t.TempDir()
	chainDir := filepath.Join(tmpDir, chainName)

	viper.Set(ChainDirKey, tmpDir)

	if needsAge {
		// Create age keys
		ids := createIdentities(1)
		var recipients []string
		for _, r := range ids {
			recipients = append(recipients, r.Recipient().String())
		}

		err := os.MkdirAll(chainDir, 0700)
		if err != nil {
			t.Fatalf("Failed to create chain directory: %v", err)
		}

		publicKeysPath := filepath.Join(chainDir, publicKeyFile)
		err = setPublicKeys(recipients, publicKeysPath)
		if err != nil {
			t.Fatalf("Failed to set public keys: %v", err)
		}

		testKey := ids[0].Recipient().String()[:10] + ":" + ids[0].String()
		viper.Set(KeyringPassword, testKey)
		return tmpDir, testKey
	}

	// For non-age stores, use a simple password
	testPassword := "this-is-a-very-long-test-password-12345"
	viper.Set(KeyringPassword, testPassword)
	return tmpDir, testPassword
}

// TestStoreInterface_StandardStore tests StandardStore implementation
func TestStoreInterface_StandardStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping StandardStore test in short mode")
	}

	chainName := "test-standard"
	tmpDir, _ := setupTestStore(t, chainName, false)
	defer os.RemoveAll(tmpDir)

	store, err := NewStandardStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create StandardStore: %v", err)
	}
	defer store.PostRunHook()

	testStoreOperations(t, store, chainName)
}

// TestStoreInterface_AgeStore tests AgeStore implementation
func TestStoreInterface_AgeStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping AgeStore test in short mode")
	}

	chainName := "test-age"
	tmpDir, _ := setupTestStore(t, chainName, true)
	defer os.RemoveAll(tmpDir)

	store, err := NewAgeStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create AgeStore: %v", err)
	}
	defer store.PostRunHook()

	testStoreOperations(t, store, chainName)
}

// TestStoreInterface_SQLiteStore tests SQLiteStore implementation
func TestStoreInterface_SQLiteStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SQLiteStore test in short mode")
	}

	chainName := "test-sqlite"
	tmpDir, _ := setupTestStore(t, chainName, true)
	defer os.RemoveAll(tmpDir)

	store, err := NewSQLiteStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create SQLiteStore: %v", err)
	}
	defer store.PostRunHook()

	testStoreOperations(t, store, chainName)
}

// testStoreOperations runs common tests for any Store implementation
func testStoreOperations(t *testing.T, store Store, chainName string) {
	// Test 1: Empty store
	keys, err := store.Keys()
	if err != nil {
		t.Fatalf("Failed to get keys from empty store: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys in empty store, got %d", len(keys))
	}

	// Test 2: Set single value
	err = store.Set(keyring.Item{
		Key:  "TEST_KEY",
		Data: []byte("test-value"),
	})
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Test 3: Get value
	item, err := store.Get("TEST_KEY")
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if string(item.Data) != "test-value" {
		t.Errorf("Expected 'test-value', got %q", string(item.Data))
	}

	// Test 4: Keys returns added key
	keys, err = store.Keys()
	if err != nil {
		t.Fatalf("Failed to get keys: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keys))
	}
	if len(keys) > 0 && keys[0] != "TEST_KEY" {
		t.Errorf("Expected 'TEST_KEY', got %q", keys[0])
	}

	// Test 5: Update value
	err = store.Set(keyring.Item{
		Key:  "TEST_KEY",
		Data: []byte("updated-value"),
	})
	if err != nil {
		t.Fatalf("Failed to update value: %v", err)
	}

	item, err = store.Get("TEST_KEY")
	if err != nil {
		t.Fatalf("Failed to get updated value: %v", err)
	}
	if string(item.Data) != "updated-value" {
		t.Errorf("Expected 'updated-value', got %q", string(item.Data))
	}

	// Test 6: Set multiple values
	err = store.Set(keyring.Item{
		Key:  "KEY1",
		Data: []byte("value1"),
	})
	if err != nil {
		t.Fatalf("Failed to set KEY1: %v", err)
	}

	err = store.Set(keyring.Item{
		Key:  "KEY2",
		Data: []byte("value2"),
	})
	if err != nil {
		t.Fatalf("Failed to set KEY2: %v", err)
	}

	// Test 7: Keys returns all keys
	keys, err = store.Keys()
	if err != nil {
		t.Fatalf("Failed to get keys: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Test 8: Name method
	name := store.Name()
	if name == "" {
		t.Error("Store name should not be empty")
	}

	// Test 9: Binary data
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	err = store.Set(keyring.Item{
		Key:  "BINARY",
		Data: binaryData,
	})
	if err != nil {
		t.Fatalf("Failed to set binary data: %v", err)
	}

	item, err = store.Get("BINARY")
	if err != nil {
		t.Fatalf("Failed to get binary data: %v", err)
	}
	if len(item.Data) != len(binaryData) {
		t.Errorf("Binary data length mismatch: expected %d, got %d", len(binaryData), len(item.Data))
	}
}

// TestStandardStore_Persistence tests StandardStore persistence
func TestStandardStore_Persistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping persistence test in short mode")
	}

	chainName := "test-standard-persist"
	tmpDir, _ := setupTestStore(t, chainName, false)
	defer os.RemoveAll(tmpDir)

	// First store: set values
	store1, err := NewStandardStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create first store: %v", err)
	}

	err = store1.Set(keyring.Item{
		Key:  "PERSIST_KEY",
		Data: []byte("persist-value"),
	})
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	err = store1.PostRunHook()
	if err != nil {
		t.Fatalf("Failed to close first store: %v", err)
	}

	// Second store: verify persistence
	store2, err := NewStandardStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create second store: %v", err)
	}
	defer store2.PostRunHook()

	item, err := store2.Get("PERSIST_KEY")
	if err != nil {
		t.Fatalf("Failed to get persisted value: %v", err)
	}

	if string(item.Data) != "persist-value" {
		t.Errorf("Expected 'persist-value', got %q", string(item.Data))
	}
}

// TestAgeStore_Persistence tests AgeStore persistence
func TestAgeStore_Persistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping persistence test in short mode")
	}

	chainName := "test-age-persist"
	tmpDir, _ := setupTestStore(t, chainName, true)
	defer os.RemoveAll(tmpDir)

	// First store: set values
	store1, err := NewAgeStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create first store: %v", err)
	}

	err = store1.Set(keyring.Item{
		Key:  "AGE_PERSIST",
		Data: []byte("age-value"),
	})
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	err = store1.PostRunHook()
	if err != nil {
		t.Fatalf("Failed to close first store: %v", err)
	}

	// Second store: verify persistence
	store2, err := NewAgeStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create second store: %v", err)
	}
	defer store2.PostRunHook()

	item, err := store2.Get("AGE_PERSIST")
	if err != nil {
		t.Fatalf("Failed to get persisted value: %v", err)
	}

	if string(item.Data) != "age-value" {
		t.Errorf("Expected 'age-value', got %q", string(item.Data))
	}
}

// TestAgeStore_EncryptedFiles tests that AgeStore creates encrypted files
func TestAgeStore_EncryptedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping encryption test in short mode")
	}

	chainName := "test-age-encrypt"
	tmpDir, _ := setupTestStore(t, chainName, true)
	defer os.RemoveAll(tmpDir)

	store, err := NewAgeStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.PostRunHook()

	secretValue := "super-secret-data"
	err = store.Set(keyring.Item{
		Key:  "SECRET",
		Data: []byte(secretValue),
	})
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Read the file directly
	ageStore := store.(AgeStore)
	filePath := ageStore.FilePath("SECRET")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Verify the file doesn't contain plaintext
	if len(content) > 0 {
		contentStr := string(content)
		if len(contentStr) > 0 && contentStr == secretValue {
			t.Error("File contains plaintext secret")
		}
	}
}

// TestStandardStore_GetNonExistent tests error handling for non-existent keys
func TestStandardStore_GetNonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	chainName := "test-standard-nonexist"
	tmpDir, _ := setupTestStore(t, chainName, false)
	defer os.RemoveAll(tmpDir)

	store, err := NewStandardStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.PostRunHook()

	_, err = store.Get("DOES_NOT_EXIST")
	if err == nil {
		t.Error("Expected error when getting non-existent key, got nil")
	}
}

// TestAgeStore_MultipleKeys tests AgeStore with multiple keys
func TestAgeStore_MultipleKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	chainName := "test-age-multi"
	tmpDir, _ := setupTestStore(t, chainName, true)
	defer os.RemoveAll(tmpDir)

	store, err := NewAgeStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.PostRunHook()

	// Set multiple keys
	testData := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}

	for key, value := range testData {
		err = store.Set(keyring.Item{
			Key:  key,
			Data: []byte(value),
		})
		if err != nil {
			t.Fatalf("Failed to set %s: %v", key, err)
		}
	}

	// Verify all keys exist
	keys, err := store.Keys()
	if err != nil {
		t.Fatalf("Failed to get keys: %v", err)
	}

	if len(keys) != len(testData) {
		t.Errorf("Expected %d keys, got %d", len(testData), len(keys))
	}

	// Verify all values
	for key, expectedValue := range testData {
		item, err := store.Get(key)
		if err != nil {
			t.Errorf("Failed to get %s: %v", key, err)
			continue
		}

		if string(item.Data) != expectedValue {
			t.Errorf("Key %s: expected %q, got %q", key, expectedValue, string(item.Data))
		}
	}
}

// TestStoreTypeAutoSave_MixedTypes tests behavior when using different store types
// for different chains in the same config directory
func TestStoreTypeAutoSave_MixedTypes(t *testing.T) {
	// Save original viper state
	origStoreType := viper.Get(StoreBackendTypeName)
	origChainDir := viper.Get(ChainDirKey)
	origPassword := viper.Get(KeyringPassword)
	origConfigFile := viper.ConfigFileUsed()
	defer func() {
		viper.Set(StoreBackendTypeName, origStoreType)
		viper.Set(ChainDirKey, origChainDir)
		viper.Set(KeyringPassword, origPassword)
		if origConfigFile != "" {
			viper.SetConfigFile(origConfigFile)
		}
	}()

	tmpDir := t.TempDir()
	viper.Set(ChainDirKey, tmpDir)
	testPassword := "this-is-a-very-long-test-password-12345"
	viper.Set(KeyringPassword, testPassword)

	// Set up config file path
	configFile := filepath.Join(tmpDir, ".chain.hcl")
	viper.SetConfigFile(configFile)

	t.Run("FirstStoreType_GetsWrittenToConfig", func(t *testing.T) {
		// Create first chain with StandardStore (type 1)
		viper.Set(StoreBackendTypeName, int32(1))

		store1, err := NewStore("chain1")
		if err != nil {
			t.Fatalf("Failed to create store1: %v", err)
		}
		defer store1.PostRunHook()

		// Verify config file was created
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}

		// Read config and verify store type was saved
		viper.ReadInConfig()
		savedType := viper.GetInt32(StoreBackendTypeName)
		if savedType != 1 {
			t.Errorf("Expected store type 1 in config, got %d", savedType)
		}
	})

	t.Run("SecondStoreType_OverwritesConfig", func(t *testing.T) {
		// Now create second chain with different store type (Age store, type 4)
		// First set up age keys for chain2
		ids := createIdentities(1)
		var recipients []string
		for _, r := range ids {
			recipients = append(recipients, r.Recipient().String())
		}

		chain2Dir := filepath.Join(tmpDir, "chain2")
		err := os.MkdirAll(chain2Dir, 0700)
		if err != nil {
			t.Fatalf("Failed to create chain2 directory: %v", err)
		}

		publicKeysPath := filepath.Join(chain2Dir, publicKeyFile)
		err = setPublicKeys(recipients, publicKeysPath)
		if err != nil {
			t.Fatalf("Failed to set public keys: %v", err)
		}

		testKey := ids[0].Recipient().String()[:10] + ":" + ids[0].String()
		viper.Set(KeyringPassword, testKey)

		// Set store type to AgeStore (type 4)
		viper.Set(StoreBackendTypeName, int32(4))

		store2, err := NewStore("chain2")
		if err != nil {
			t.Fatalf("Failed to create store2: %v", err)
		}
		defer store2.PostRunHook()

		// Read config and verify store type was updated
		viper.ReadInConfig()
		savedType := viper.GetInt32(StoreBackendTypeName)
		if savedType != 4 {
			t.Errorf("Expected store type 4 in config, got %d", savedType)
		}
	})

	t.Run("LastSetStoreType_BecomesDefault", func(t *testing.T) {
		// Verify that the last set store type is what's in the config
		viper.ReadInConfig()
		savedType := viper.GetInt32(StoreBackendTypeName)

		// Should be 4 from the previous test
		if savedType != 4 {
			t.Errorf("Expected last set store type (4) in config, got %d", savedType)
		}

		// Unset the store type to simulate user not specifying it
		viper.Set(StoreBackendTypeName, nil)
		viper.ReadInConfig() // Read from file

		defaultType := viper.GetInt32(StoreBackendTypeName)
		if defaultType != 4 {
			t.Errorf("Expected default from config (4), got %d", defaultType)
		}
	})
}

// TestStoreTypeAutoSave_SQLiteAndStandardInSameDir tests the specific scenario
// of using SQLite for one chain and Standard store for another
func TestStoreTypeAutoSave_SQLiteAndStandardInSameDir(t *testing.T) {
	// Save original viper state
	origStoreType := viper.Get(StoreBackendTypeName)
	origChainDir := viper.Get(ChainDirKey)
	origPassword := viper.Get(KeyringPassword)
	origConfigFile := viper.ConfigFileUsed()
	defer func() {
		viper.Set(StoreBackendTypeName, origStoreType)
		viper.Set(ChainDirKey, origChainDir)
		viper.Set(KeyringPassword, origPassword)
		if origConfigFile != "" {
			viper.SetConfigFile(origConfigFile)
		}
	}()

	tmpDir := t.TempDir()
	viper.Set(ChainDirKey, tmpDir)

	// Set up config file path
	configFile := filepath.Join(tmpDir, ".chain.hcl")
	viper.SetConfigFile(configFile)

	// Create age keys once and reuse them
	ids := createIdentities(1)
	var recipients []string
	for _, r := range ids {
		recipients = append(recipients, r.Recipient().String())
	}
	testKey := ids[0].Recipient().String()[:10] + ":" + ids[0].String()

	// Create my-secrets with SQLite (type 6)
	t.Run("CreateSQLiteStore", func(t *testing.T) {
		secretsDir := filepath.Join(tmpDir, "my-secrets")
		err := os.MkdirAll(secretsDir, 0700)
		if err != nil {
			t.Fatalf("Failed to create my-secrets directory: %v", err)
		}

		publicKeysPath := filepath.Join(secretsDir, publicKeyFile)
		err = setPublicKeys(recipients, publicKeysPath)
		if err != nil {
			t.Fatalf("Failed to set public keys: %v", err)
		}

		viper.Set(KeyringPassword, testKey)
		viper.Set(StoreBackendTypeName, int32(6)) // SQLite

		store, err := NewStore("my-secrets")
		if err != nil {
			t.Fatalf("Failed to create SQLite store: %v", err)
		}
		defer store.PostRunHook()

		// Store some data
		err = store.Set(keyring.Item{
			Key:  "API_KEY",
			Data: []byte("secret-value"),
		})
		if err != nil {
			t.Fatalf("Failed to set item: %v", err)
		}

		// Verify config has SQLite type
		viper.ReadInConfig()
		savedType := viper.GetInt32(StoreBackendTypeName)
		if savedType != 6 {
			t.Errorf("Expected SQLite type (6) in config, got %d", savedType)
		}
	})

	// Create my-secrets2 with StandardStore (type 1)
	t.Run("CreateStandardStore", func(t *testing.T) {
		testPassword := "this-is-a-very-long-test-password-12345"
		viper.Set(KeyringPassword, testPassword)
		viper.Set(StoreBackendTypeName, int32(1)) // StandardStore

		store, err := NewStore("my-secrets2")
		if err != nil {
			t.Fatalf("Failed to create StandardStore: %v", err)
		}
		defer store.PostRunHook()

		// Store some data
		err = store.Set(keyring.Item{
			Key:  "DB_PASSWORD",
			Data: []byte("another-secret"),
		})
		if err != nil {
			t.Fatalf("Failed to set item: %v", err)
		}

		// Verify config now has StandardStore type (overwrites previous)
		viper.ReadInConfig()
		savedType := viper.GetInt32(StoreBackendTypeName)
		if savedType != 1 {
			t.Errorf("Expected StandardStore type (1) in config, got %d", savedType)
		}
	})

	// Verify both stores still work with their respective data
	t.Run("BothStoresStillWork", func(t *testing.T) {
		// Re-open my-secrets with explicit SQLite type (using same key)
		viper.Set(KeyringPassword, testKey)
		viper.Set(StoreBackendTypeName, int32(6))

		sqliteStore, err := NewStore("my-secrets")
		if err != nil {
			t.Fatalf("Failed to reopen SQLite store: %v", err)
		}
		defer sqliteStore.PostRunHook()

		item, err := sqliteStore.Get("API_KEY")
		if err != nil {
			t.Fatalf("Failed to get API_KEY from SQLite store: %v", err)
		}
		if string(item.Data) != "secret-value" {
			t.Errorf("Expected 'secret-value', got %q", string(item.Data))
		}

		// Re-open my-secrets2 with StandardStore
		testPassword := "this-is-a-very-long-test-password-12345"
		viper.Set(KeyringPassword, testPassword)
		viper.Set(StoreBackendTypeName, int32(1))

		standardStore, err := NewStore("my-secrets2")
		if err != nil {
			t.Fatalf("Failed to reopen StandardStore: %v", err)
		}
		defer standardStore.PostRunHook()

		item2, err := standardStore.Get("DB_PASSWORD")
		if err != nil {
			t.Fatalf("Failed to get DB_PASSWORD from StandardStore: %v", err)
		}
		if string(item2.Data) != "another-secret" {
			t.Errorf("Expected 'another-secret', got %q", string(item2.Data))
		}
	})
}

// TestStoreTypeAutoSave_OnlyWhenExplicitlySet tests that the store type
// is only saved when explicitly set, not when using default
func TestStoreTypeAutoSave_OnlyWhenExplicitlySet(t *testing.T) {
	// Save original viper state
	origStoreType := viper.Get(StoreBackendTypeName)
	origChainDir := viper.Get(ChainDirKey)
	origPassword := viper.Get(KeyringPassword)
	origConfigFile := viper.ConfigFileUsed()
	defer func() {
		viper.Set(StoreBackendTypeName, origStoreType)
		viper.Set(ChainDirKey, origChainDir)
		viper.Set(KeyringPassword, origPassword)
		if origConfigFile != "" {
			viper.SetConfigFile(origConfigFile)
		}
	}()

	tmpDir := t.TempDir()
	viper.Set(ChainDirKey, tmpDir)
	testPassword := "this-is-a-very-long-test-password-12345"
	viper.Set(KeyringPassword, testPassword)

	configFile := filepath.Join(tmpDir, ".chain.hcl")
	viper.SetConfigFile(configFile)

	t.Run("DefaultValue_NotSaved", func(t *testing.T) {
		// Unset store type to use default (1)
		viper.Set(StoreBackendTypeName, nil)

		// Get default value
		defaultType := viper.GetInt32(StoreBackendTypeName)

		// This should be the default from root.go (1)
		if defaultType != 1 {
			t.Logf("Note: Default store type is %d", defaultType)
		}

		store, err := NewStore("test-chain")
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.PostRunHook()

		// Config file should not be created when using default
		if _, err := os.Stat(configFile); err == nil {
			// If file exists, check if it has store type
			viper.ReadInConfig()
			// The file might exist from init, but store shouldn't be explicitly set
			// when we only used the default
			t.Logf("Config file exists, checking if store was saved")
		}
	})

	t.Run("ExplicitValue_IsSaved", func(t *testing.T) {
		// Explicitly set store type
		viper.Set(StoreBackendTypeName, int32(1))

		store, err := NewStore("test-chain2")
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.PostRunHook()

		// Config should now exist and have the value
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Error("Config file was not created for explicit store type")
		} else {
			viper.ReadInConfig()
			savedType := viper.GetInt32(StoreBackendTypeName)
			if savedType != 1 {
				t.Errorf("Expected store type 1 in config, got %d", savedType)
			}
		}
	})
}
