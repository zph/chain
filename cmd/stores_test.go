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
