package cmd

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/99designs/keyring"
	"github.com/spf13/viper"
	_ "modernc.org/sqlite"
)

func TestSQLiteStore_EncryptsColumns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping encryption test in short mode")
	}

	tmpDir := t.TempDir()
	chainName := "test-chain"
	chainDir := filepath.Join(tmpDir, chainName)

	// Set up viper config
	viper.Set(ChainDirKey, tmpDir)

	// Create age keys
	ids := createIdentities(2)
	var recipients []string
	for _, r := range ids {
		recipients = append(recipients, r.Recipient().String())
	}

	// Write public keys
	err := os.MkdirAll(chainDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create chain directory: %v", err)
	}

	publicKeysPath := filepath.Join(chainDir, publicKeyFile)
	err = setPublicKeys(recipients, publicKeysPath)
	if err != nil {
		t.Fatalf("Failed to set public keys: %v", err)
	}

	// Set password with key format
	testKey := ids[0].Recipient().String()[:10] + ":" + ids[0].String()
	viper.Set(KeyringPassword, testKey)

	// Create store
	store, err := NewSQLiteStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.PostRunHook()

	// Set a value
	secretValue := "super-secret-value-12345"
	secretKey := "SECRET_KEY"
	err = store.Set(keyring.Item{
		Key:  secretKey,
		Data: []byte(secretValue),
	})
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Close the store to ensure data is flushed
	store.PostRunHook()

	// Open database directly and verify data is encrypted
	dbPath := filepath.Join(chainDir, "chain.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database directly: %v", err)
	}
	defer db.Close()

	// Query variables table - value should be encrypted
	rows, err := db.Query("SELECT id, value_encrypted, collection_hash, collection_encrypted FROM variables")
	if err != nil {
		t.Fatalf("Failed to query variables: %v", err)
	}
	defer rows.Close()

	foundVar := false
	for rows.Next() {
		var id, collectionHash string
		var valueEnc, collectionEnc []byte
		if err := rows.Scan(&id, &valueEnc, &collectionHash, &collectionEnc); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		foundVar = true

		// Verify value is encrypted (should contain age header or be binary)
		valueStr := string(valueEnc)
		if strings.Contains(valueStr, secretValue) {
			t.Error("value_encrypted column contains plaintext secret")
		}

		// Verify collection is encrypted
		collectionStr := string(collectionEnc)
		if strings.Contains(collectionStr, chainName) {
			t.Error("collection_encrypted column contains plaintext collection name")
		}

		// Verify collection_hash doesn't contain plaintext
		if strings.Contains(collectionHash, chainName) {
			t.Error("collection_hash contains plaintext collection name")
		}
	}

	if !foundVar {
		t.Error("No variables found in database")
	}

	// Query name_mapping table - name, salt, and pepper should be encrypted
	rows, err = db.Query("SELECT name_hash, salt_encrypted, pepper_encrypted, name_encrypted, collection_hash, collection_encrypted FROM name_mapping")
	if err != nil {
		t.Fatalf("Failed to query name_mapping: %v", err)
	}
	defer rows.Close()

	foundMapping := false
	for rows.Next() {
		var nameHash, collectionHash string
		var saltEnc, pepperEnc, nameEnc, collectionEnc []byte
		if err := rows.Scan(&nameHash, &saltEnc, &pepperEnc, &nameEnc, &collectionHash, &collectionEnc); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		foundMapping = true

		// Verify name is encrypted
		nameStr := string(nameEnc)
		if strings.Contains(nameStr, secretKey) {
			t.Error("name_encrypted column contains plaintext name")
		}

		// Verify salt is encrypted (should not be readable hex)
		saltStr := string(saltEnc)
		if len(saltStr) == 64 && isHexString(saltStr) {
			t.Error("salt_encrypted appears to be plaintext hex")
		}

		// Verify pepper is encrypted (should not be readable hex)
		pepperStr := string(pepperEnc)
		if len(pepperStr) == 64 && isHexString(pepperStr) {
			t.Error("pepper_encrypted appears to be plaintext hex")
		}

		// Verify collection is encrypted
		collectionStr := string(collectionEnc)
		if strings.Contains(collectionStr, chainName) {
			t.Error("collection_encrypted in name_mapping contains plaintext")
		}

		// Verify collection_hash doesn't contain plaintext
		if strings.Contains(collectionHash, chainName) {
			t.Error("collection_hash in name_mapping contains plaintext collection name")
		}
	}

	if !foundMapping {
		t.Error("No name mappings found in database")
	}
}

// isHexString checks if a string is valid hexadecimal
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func TestSQLiteStore_MultipleValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	tmpDir := t.TempDir()
	chainName := "test-chain"
	chainDir := filepath.Join(tmpDir, chainName)

	viper.Set(ChainDirKey, tmpDir)

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

	store, err := NewSQLiteStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.PostRunHook()

	// Set multiple values
	items := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}

	for key, value := range items {
		err = store.Set(keyring.Item{
			Key:  key,
			Data: []byte(value),
		})
		if err != nil {
			t.Fatalf("Failed to set %s: %v", key, err)
		}
	}

	// Get all values
	for key, expectedValue := range items {
		item, err := store.Get(key)
		if err != nil {
			t.Errorf("Failed to get %s: %v", key, err)
			continue
		}

		if string(item.Data) != expectedValue {
			t.Errorf("Key %s: expected %q, got %q", key, expectedValue, string(item.Data))
		}
	}

	// Verify Keys() returns all keys
	keys, err := store.Keys()
	if err != nil {
		t.Fatalf("Failed to get keys: %v", err)
	}

	if len(keys) != len(items) {
		t.Errorf("Expected %d keys, got %d", len(items), len(keys))
	}
}

func TestSQLiteStore_Remove(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	tmpDir := t.TempDir()
	chainName := "test-chain"
	chainDir := filepath.Join(tmpDir, chainName)

	viper.Set(ChainDirKey, tmpDir)

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

	store, err := NewSQLiteStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.PostRunHook()

	// Set a value
	err = store.Set(keyring.Item{
		Key:  "TEST_KEY",
		Data: []byte("test-value"),
	})
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Remove it
	err = store.Remove("TEST_KEY")
	if err != nil {
		t.Fatalf("Failed to remove value: %v", err)
	}

	// Verify it's gone
	_, err = store.Get("TEST_KEY")
	if err == nil {
		t.Error("Expected error when getting removed key, got nil")
	}
}

func TestSQLiteStore_RemoveNonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	tmpDir := t.TempDir()
	chainName := "test-chain"
	chainDir := filepath.Join(tmpDir, chainName)

	viper.Set(ChainDirKey, tmpDir)

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

	store, err := NewSQLiteStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.PostRunHook()

	// Try to remove non-existent key
	err = store.Remove("DOES_NOT_EXIST")
	if err == nil {
		t.Error("Expected error when removing non-existent key, got nil")
	}
}

func TestSQLiteStore_UpdateValue(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	tmpDir := t.TempDir()
	chainName := "test-chain"
	chainDir := filepath.Join(tmpDir, chainName)

	viper.Set(ChainDirKey, tmpDir)

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

	store, err := NewSQLiteStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.PostRunHook()

	// Set initial value
	err = store.Set(keyring.Item{
		Key:  "UPDATE_KEY",
		Data: []byte("initial-value"),
	})
	if err != nil {
		t.Fatalf("Failed to set initial value: %v", err)
	}

	// Update value
	err = store.Set(keyring.Item{
		Key:  "UPDATE_KEY",
		Data: []byte("updated-value"),
	})
	if err != nil {
		t.Fatalf("Failed to update value: %v", err)
	}

	// Get updated value
	item, err := store.Get("UPDATE_KEY")
	if err != nil {
		t.Fatalf("Failed to get updated value: %v", err)
	}

	if string(item.Data) != "updated-value" {
		t.Errorf("Expected 'updated-value', got %q", string(item.Data))
	}
}

func TestSQLiteStore_GetNonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	tmpDir := t.TempDir()
	chainName := "test-chain"
	chainDir := filepath.Join(tmpDir, chainName)

	viper.Set(ChainDirKey, tmpDir)

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

	store, err := NewSQLiteStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.PostRunHook()

	_, err = store.Get("DOES_NOT_EXIST")
	if err == nil {
		t.Error("Expected error when getting non-existent key, got nil")
	}
}

func TestSQLiteStore_KeysEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	tmpDir := t.TempDir()
	chainName := "test-chain"
	chainDir := filepath.Join(tmpDir, chainName)

	viper.Set(ChainDirKey, tmpDir)

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

	store, err := NewSQLiteStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.PostRunHook()

	keys, err := store.Keys()
	if err != nil {
		t.Fatalf("Failed to get keys: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys in empty store, got %d", len(keys))
	}
}

func TestSQLiteStore_BinaryData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	tmpDir := t.TempDir()
	chainName := "test-chain"
	chainDir := filepath.Join(tmpDir, chainName)

	viper.Set(ChainDirKey, tmpDir)

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

	store, err := NewSQLiteStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.PostRunHook()

	// Set binary data
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	err = store.Set(keyring.Item{
		Key:  "BINARY",
		Data: binaryData,
	})
	if err != nil {
		t.Fatalf("Failed to set binary data: %v", err)
	}

	// Get binary data
	item, err := store.Get("BINARY")
	if err != nil {
		t.Fatalf("Failed to get binary data: %v", err)
	}

	if len(item.Data) != len(binaryData) {
		t.Errorf("Binary data length mismatch: expected %d, got %d", len(binaryData), len(item.Data))
	}

	for i := range binaryData {
		if item.Data[i] != binaryData[i] {
			t.Errorf("Binary data mismatch at byte %d: expected %02x, got %02x", i, binaryData[i], item.Data[i])
		}
	}
}

func TestSQLiteStore_Name(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	tmpDir := t.TempDir()
	chainName := "test-chain"
	chainDir := filepath.Join(tmpDir, chainName)

	viper.Set(ChainDirKey, tmpDir)

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

	store, err := NewSQLiteStore(chainName)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.PostRunHook()

	name := store.Name()
	if name == "" {
		t.Error("Store name should not be empty")
	}
}
