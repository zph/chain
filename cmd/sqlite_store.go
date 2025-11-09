package cmd

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/99designs/keyring"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	chainv1 "github.com/zph/chain/gen/go/chain/v1"
	_ "modernc.org/sqlite"
	"golang.org/x/crypto/argon2"
)

type SQLiteStore struct {
	db         *sql.DB
	collection string
	Config     keyring.Config
	dbFilePath string
	recipients []age.Recipient
	identity   age.Identity
}

func NewSQLiteStore(chain string) (Store, error) {
	// Ensure the directory exists
	dirPath := filePath(chain)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return nil, eris.Wrap(err, "failed to create directory for SQLite store")
	}

	cfg := keyring.Config{
		ServiceName:      chain,
		FilePasswordFunc: getPassword,
		FileDir:          dirPath,
	}

	// Path to on-disk database file
	dbFilePath := filepath.Join(dirPath, "chain.db")

	// Open on-disk database
	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open SQLite database")
	}

	// Create schema with encrypted columns
	schema := `
	CREATE TABLE IF NOT EXISTS variables (
		id TEXT PRIMARY KEY,
		value_encrypted BLOB NOT NULL,
		collection_hash TEXT NOT NULL,
		collection_encrypted BLOB NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_collection_hash ON variables(collection_hash);

	CREATE TABLE IF NOT EXISTS name_mapping (
		name_hash TEXT NOT NULL,
		id TEXT NOT NULL,
		salt_encrypted BLOB NOT NULL,
		pepper_encrypted BLOB NOT NULL,
		name_encrypted BLOB NOT NULL,
		collection_hash TEXT NOT NULL,
		collection_encrypted BLOB NOT NULL,
		PRIMARY KEY (name_hash, collection_hash),
		FOREIGN KEY (id) REFERENCES variables(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_name_hash ON name_mapping(name_hash);
	CREATE INDEX IF NOT EXISTS idx_collection_hash_nm ON name_mapping(collection_hash);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, eris.Wrap(err, "failed to create schema")
	}

	store := &SQLiteStore{
		db:         db,
		collection: chain,
		Config:     cfg,
		dbFilePath: dbFilePath,
	}

	// Load age keys
	if err := store.loadAgeKeys(); err != nil {
		db.Close()
		return nil, eris.Wrap(err, "failed to load age keys")
	}

	log.Debug().Str("path", dbFilePath).Str("collection", chain).Msg("SQLite store initialized")

	return store, nil
}

// loadAgeKeys loads the age encryption keys
func (s *SQLiteStore) loadAgeKeys() error {
	publicKeyPrefix, privateKey, err := s.getKeysFromUser()
	if err != nil {
		return eris.Wrap(err, "failed to get keys from user")
	}

	identity, err := age.ParseX25519Identity(strings.TrimSpace(privateKey))
	if err != nil {
		return eris.Wrap(err, "failed to parse private key")
	}
	s.identity = identity

	// Load public keys
	publicKeysPath := filepath.Join(s.Config.FileDir, publicKeyFile)
	publicKeys, err := os.Open(publicKeysPath)
	if err != nil {
		return eris.Wrap(err, "failed to open public keys file")
	}
	defer publicKeys.Close()

	recipients, err := age.ParseRecipients(publicKeys)
	if err != nil {
		return eris.Wrap(err, "failed to parse public keys")
	}
	s.recipients = recipients

	_ = publicKeyPrefix // not used but returned from getKeysFromUser
	return nil
}

// encrypt encrypts data using age
func (s *SQLiteStore) encrypt(data []byte) ([]byte, error) {
	out := &bytes.Buffer{}
	w, err := age.Encrypt(out, s.recipients...)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create encrypted writer")
	}

	if _, err := w.Write(data); err != nil {
		return nil, eris.Wrap(err, "failed to write encrypted data")
	}

	if err := w.Close(); err != nil {
		return nil, eris.Wrap(err, "failed to close encrypted writer")
	}

	return out.Bytes(), nil
}

// decrypt decrypts data using age
func (s *SQLiteStore) decrypt(data []byte) ([]byte, error) {
	r, err := age.Decrypt(bytes.NewReader(data), s.identity)
	if err != nil {
		return nil, eris.Wrap(err, "failed to decrypt data")
	}

	plaintext, err := io.ReadAll(r)
	if err != nil {
		return nil, eris.Wrap(err, "failed to read decrypted data")
	}

	return plaintext, nil
}

// generateSaltPepper generates a new 32-byte salt and pepper
func generateSaltPepper() (string, string, error) {
	salt := make([]byte, 32)
	pepper := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return "", "", eris.Wrap(err, "failed to generate salt")
	}
	if _, err := rand.Read(pepper); err != nil {
		return "", "", eris.Wrap(err, "failed to generate pepper")
	}
	return hex.EncodeToString(salt), hex.EncodeToString(pepper), nil
}

// hashName creates a memory-hard hash of the name using Argon2id with salt and pepper
func hashName(name, salt, pepper string) (string, error) {
	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		return "", eris.Wrap(err, "failed to decode salt")
	}
	pepperBytes, err := hex.DecodeString(pepper)
	if err != nil {
		return "", eris.Wrap(err, "failed to decode pepper")
	}

	// Argon2id parameters (same as collection hash for consistency)
	const (
		timeCost    = 1
		memoryCost  = 64 * 1024 // 64 MB
		parallelism = 4
		keyLength   = 32
	)

	// Combine salt and pepper for the Argon2 salt parameter
	// This ensures unique salt per variable
	combinedSalt := append(saltBytes, pepperBytes...)

	// Use SHA256 to ensure salt is exactly 32 bytes (Argon2 recommends at least 16 bytes)
	finalSalt := sha256.Sum256(combinedSalt)

	hash := argon2.IDKey([]byte(name), finalSalt[:], timeCost, memoryCost, parallelism, keyLength)
	return hex.EncodeToString(hash), nil
}

// hashCollection creates a memory-hard hash of the collection name using Argon2id
// This makes brute-forcing collection names computationally expensive
func hashCollection(collection string) string {
	// Argon2id parameters (balanced for security and performance)
	// time=1, memory=64MB, threads=4, keyLen=32
	// These are OWASP recommended minimums for password hashing
	const (
		timeCost    = 1
		memoryCost  = 64 * 1024 // 64 MB
		parallelism = 4
		keyLength   = 32
	)

	// Use a fixed salt derived from the collection name
	// This makes the hash deterministic for the same collection
	// We need deterministic hashes so we can query by collection_hash
	salt := sha256.Sum256([]byte("chain-collection-salt-v1:" + collection))

	hash := argon2.IDKey([]byte(collection), salt[:], timeCost, memoryCost, parallelism, keyLength)
	return hex.EncodeToString(hash)
}

// lookupID finds the ID for a given name
func (s *SQLiteStore) lookupID(name string) (string, string, string, error) {
	collectionHash := hashCollection(s.collection)

	// Query by collection_hash
	rows, err := s.db.Query(`
		SELECT id, salt_encrypted, pepper_encrypted, name_encrypted FROM name_mapping
		WHERE collection_hash = ?
	`, collectionHash)
	if err != nil {
		return "", "", "", eris.Wrap(err, "failed to query name_mapping")
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var saltEnc, pepperEnc, nameEnc []byte
		if err := rows.Scan(&id, &saltEnc, &pepperEnc, &nameEnc); err != nil {
			return "", "", "", eris.Wrap(err, "failed to scan row")
		}

		// Decrypt name and check if this is the right one
		decryptedName, err := s.decrypt(nameEnc)
		if err != nil {
			continue
		}

		if string(decryptedName) == name {
			// Decrypt salt and pepper
			saltBytes, err := s.decrypt(saltEnc)
			if err != nil {
				return "", "", "", eris.Wrap(err, "failed to decrypt salt")
			}
			pepperBytes, err := s.decrypt(pepperEnc)
			if err != nil {
				return "", "", "", eris.Wrap(err, "failed to decrypt pepper")
			}

			return id, string(saltBytes), string(pepperBytes), nil
		}
	}

	return "", "", "", eris.New(fmt.Sprintf("key %q not found", name))
}

func (s *SQLiteStore) Keys() ([]string, error) {
	collectionHash := hashCollection(s.collection)

	rows, err := s.db.Query(`
		SELECT name_encrypted
		FROM name_mapping
		WHERE collection_hash = ?
	`, collectionHash)
	if err != nil {
		return nil, eris.Wrap(err, "failed to query keys")
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var nameEnc []byte
		if err := rows.Scan(&nameEnc); err != nil {
			return nil, eris.Wrap(err, "failed to scan name")
		}

		name, err := s.decrypt(nameEnc)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to decrypt name")
			continue
		}
		keys = append(keys, string(name))
	}

	if err := rows.Err(); err != nil {
		return nil, eris.Wrap(err, "error iterating rows")
	}

	return keys, nil
}

func (s *SQLiteStore) Get(key string) (keyring.Item, error) {
	id, _, _, err := s.lookupID(key)
	if err != nil {
		return keyring.Item{}, err
	}

	collectionHash := hashCollection(s.collection)

	var valueEnc []byte
	err = s.db.QueryRow(`
		SELECT value_encrypted FROM variables WHERE id = ? AND collection_hash = ?
	`, id, collectionHash).Scan(&valueEnc)

	if err == sql.ErrNoRows {
		return keyring.Item{}, eris.New(fmt.Sprintf("key %q not found", key))
	}
	if err != nil {
		return keyring.Item{}, eris.Wrap(err, "failed to get value")
	}

	value, err := s.decrypt(valueEnc)
	if err != nil {
		return keyring.Item{}, eris.Wrap(err, "failed to decrypt value")
	}

	return keyring.Item{
		Key:  key,
		Data: value,
	}, nil
}

func (s *SQLiteStore) Set(item keyring.Item) error {
	now := time.Now()
	collectionHash := hashCollection(s.collection)

	// Encrypt collection
	collectionEnc, err := s.encrypt([]byte(s.collection))
	if err != nil {
		return eris.Wrap(err, "failed to encrypt collection")
	}

	// Check if already exists
	existingID, salt, pepper, err := s.lookupID(item.Key)
	var itemID string

	if err != nil {
		// New item - generate nanoid and salt/pepper
		itemID, err = gonanoid.New()
		if err != nil {
			return eris.Wrap(err, "failed to generate nanoid")
		}

		salt, pepper, err = generateSaltPepper()
		if err != nil {
			return eris.Wrap(err, "failed to generate salt and pepper")
		}

		nameHash, err := hashName(item.Key, salt, pepper)
		if err != nil {
			return eris.Wrap(err, "failed to hash name")
		}

		// Encrypt value, name, salt, and pepper
		valueEnc, err := s.encrypt(item.Data)
		if err != nil {
			return eris.Wrap(err, "failed to encrypt value")
		}

		nameEnc, err := s.encrypt([]byte(item.Key))
		if err != nil {
			return eris.Wrap(err, "failed to encrypt name")
		}

		saltEnc, err := s.encrypt([]byte(salt))
		if err != nil {
			return eris.Wrap(err, "failed to encrypt salt")
		}

		pepperEnc, err := s.encrypt([]byte(pepper))
		if err != nil {
			return eris.Wrap(err, "failed to encrypt pepper")
		}

		// Insert into variables
		_, err = s.db.Exec(`
			INSERT INTO variables (id, value_encrypted, collection_hash, collection_encrypted, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, itemID, valueEnc, collectionHash, collectionEnc, now, now)
		if err != nil {
			return eris.Wrap(err, "failed to insert value")
		}

		// Insert into name_mapping with encrypted name, salt, and pepper
		_, err = s.db.Exec(`
			INSERT INTO name_mapping (name_hash, id, salt_encrypted, pepper_encrypted, name_encrypted, collection_hash, collection_encrypted)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, nameHash, itemID, saltEnc, pepperEnc, nameEnc, collectionHash, collectionEnc)
		if err != nil {
			return eris.Wrap(err, "failed to insert name mapping")
		}
	} else {
		// Update existing item
		itemID = existingID

		valueEnc, err := s.encrypt(item.Data)
		if err != nil {
			return eris.Wrap(err, "failed to encrypt value")
		}

		_, err = s.db.Exec(`
			UPDATE variables
			SET value_encrypted = ?, updated_at = ?
			WHERE id = ? AND collection_hash = ?
		`, valueEnc, now, itemID, collectionHash)
		if err != nil {
			return eris.Wrap(err, "failed to update value")
		}
	}

	log.Debug().Str("key", item.Key).Str("id", itemID).Str("collection", s.collection).Msg("Set value in SQLite store")

	return nil
}

func (s *SQLiteStore) Remove(key string) error {
	id, salt, pepper, err := s.lookupID(key)
	if err != nil {
		return err
	}

	collectionHash := hashCollection(s.collection)

	// Delete from variables (cascade will handle name_mapping)
	result, err := s.db.Exec(`DELETE FROM variables WHERE id = ? AND collection_hash = ?`, id, collectionHash)
	if err != nil {
		return eris.Wrap(err, "failed to remove value")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return eris.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return eris.New(fmt.Sprintf("key %q not found", key))
	}

	// Also remove name mapping manually (in case CASCADE doesn't work)
	nameHash, err := hashName(key, salt, pepper)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to hash name for deletion")
	} else {
		_, err = s.db.Exec(`DELETE FROM name_mapping WHERE name_hash = ? AND collection_hash = ?`, nameHash, collectionHash)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to delete name mapping")
		}
	}

	log.Debug().Str("key", key).Str("id", id).Str("collection", s.collection).Msg("Removed value from SQLite store")

	return nil
}

func (s *SQLiteStore) Name() string {
	return chainv1.StorageType_STORAGE_TYPE_SQLITE.String()
}

func (s *SQLiteStore) PostRunHook() error {
	// Close the database connection
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return eris.Wrap(err, "failed to close database")
		}
		log.Debug().Msg("Closed SQLite database connection")
	}
	return nil
}

// getKeysFromUser retrieves the age key pair from the user
func (s *SQLiteStore) getKeysFromUser() (string, string, error) {
	pk, err := s.Config.FilePasswordFunc("")
	if err != nil {
		return "", "", err
	}

	publicKeyPrefix, privateKey, found := strings.Cut(pk, ":")
	if !found {
		return "", "", errors.New("invalid key format, expected publicKeyPrefix:privateKey")
	}

	return publicKeyPrefix, privateKey, nil
}
