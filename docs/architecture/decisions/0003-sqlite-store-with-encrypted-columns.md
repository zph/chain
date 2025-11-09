# 3. SQLite Store with Encrypted Columns

Date: 2025-01-09

## Status

Accepted

## Context

We needed a new storage backend that:
1. Stores secrets in a structured database format
2. Provides encryption at rest
3. Prevents information leakage through database inspection
4. Supports fast indexed lookups
5. Remains portable and doesn't require external database servers

Previous storage options (StandardStore, AgeStore) either lacked structure or required entire-file encryption/decryption, making selective access inefficient.

## Decision

We implemented a SQLite-based storage engine with the following design choices:

### 1. On-Disk SQLite Database with Column-Level Encryption

**Decision:** Use on-disk SQLite with Age encryption applied per-column, rather than encrypting the entire database file.

**Rationale:**
- Column-level encryption allows the database to remain queryable
- SQLite's indexing capabilities work on non-encrypted columns (hashes)
- Age encryption provides strong, modern cryptography
- No plaintext data ever written to disk

### 2. Nanoid for Record Identifiers

**Decision:** Use nanoid instead of UUID for primary keys.

**Rationale:**
- Shorter and more URL-friendly than UUIDs
- Cryptographically secure random generation
- Reduces storage and index size

### 3. Argon2id for Hash Functions

**Decision:** Use Argon2id (memory-hard hashing) instead of SHA256 for both `collection_hash` and `name_hash`.

**Rationale:**
- **Memory-hard:** Requires 64MB RAM per hash attempt, making brute-force attacks ~1000x more expensive
- **GPU/ASIC resistant:** Cannot be efficiently parallelized on specialized hardware
- **OWASP recommended:** Industry standard for password hashing
- **Protection against offline attacks:** Even if database is stolen, attacker cannot efficiently guess collection/variable names

**Parameters chosen:**
```
time=1, memory=64MB, threads=4, keyLen=32
```
These are OWASP recommended minimums, balancing security with acceptable performance (~100ms per operation).

### 4. Per-Variable Salt and Pepper (Encrypted)

**Decision:** Generate unique 32-byte salt and pepper for each variable name, storing them encrypted.

**Rationale:**
- Prevents rainbow table attacks
- Each variable has unique hash even if names are identical across collections
- Encrypted salt/pepper prevents offline brute-force attacks
- Defense in depth: attacker needs both database access AND Age decryption keys

### 5. Schema Design

**Variables Table:**
```sql
CREATE TABLE variables (
    id TEXT PRIMARY KEY,                  -- nanoid
    value_encrypted BLOB NOT NULL,        -- Age-encrypted value
    collection_hash TEXT NOT NULL,        -- Argon2id hash
    collection_encrypted BLOB NOT NULL,   -- Age-encrypted collection
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

**Name Mapping Table:**
```sql
CREATE TABLE name_mapping (
    name_hash TEXT NOT NULL,              -- Argon2id(salt + name + pepper)
    id TEXT NOT NULL,                     -- references variables.id
    salt_encrypted BLOB NOT NULL,         -- Age-encrypted 32-byte salt
    pepper_encrypted BLOB NOT NULL,       -- Age-encrypted 32-byte pepper
    name_encrypted BLOB NOT NULL,         -- Age-encrypted variable name
    collection_hash TEXT NOT NULL,        -- Argon2id hash
    collection_encrypted BLOB NOT NULL,   -- Age-encrypted collection
    PRIMARY KEY (name_hash, collection_hash),
    FOREIGN KEY (id) REFERENCES variables(id) ON DELETE CASCADE
);
```

**Design rationale:**
- Separation of concerns: variables table for data, name_mapping for lookups
- Indexed hashes enable O(1) lookups without exposing names
- Encrypted names preserved for Keys() operation
- CASCADE ensures referential integrity

### 6. What is Encrypted vs. Hashed

**Encrypted (Age):**
- `value_encrypted` - Secret values
- `name_encrypted` - Variable names
- `collection_encrypted` - Collection names
- `salt_encrypted` - Per-variable salts
- `pepper_encrypted` - Per-variable peppers

**Hashed (Argon2id):**
- `collection_hash` - For indexing/lookups (deterministic)
- `name_hash` - For obfuscated lookups (unique per variable due to salt+pepper)

**Not Protected:**
- `id` - Random nanoid (no information leakage)
- Timestamps - Not sensitive

## Consequences

### Positive

1. **Strong Security:**
   - No plaintext data in database
   - Memory-hard hashing prevents brute force
   - Per-variable salts prevent rainbow tables
   - Defense in depth with encryption + hashing

2. **Performance:**
   - Indexed lookups via `collection_hash` and `name_hash`
   - No need to decrypt entire database
   - O(n) where n = variables in collection (acceptable for typical use)

3. **Structured Storage:**
   - SQL schema provides structure
   - Easy to query and maintain
   - Supports future schema migrations

4. **Portability:**
   - Single file database
   - No external server required
   - Works across platforms

### Negative

1. **Performance Cost:**
   - Argon2id adds ~100ms per hash operation
   - Lookups require decrypting all names in collection (O(n))
   - Test suite ~2x slower than SHA256 version

2. **Complexity:**
   - More complex than simple file encryption
   - Requires managing two tables with foreign keys
   - Need to decrypt multiple columns per operation

3. **Memory Usage:**
   - Argon2id requires 64MB per hash
   - Multiple concurrent operations multiply memory needs

### Risks and Mitigations

**Risk:** Argon2id parameters may become outdated
- **Mitigation:** Parameters are in constants, easy to update. Can version the schema and migrate.

**Risk:** Age encryption key management complexity
- **Mitigation:** Reuses existing Age key infrastructure from other stores.

**Risk:** Performance degradation with large collections
- **Mitigation:** O(n) lookup is acceptable for typical secret management (< 1000 variables). For larger scales, could add caching.

## Alternatives Considered

1. **Entire database encryption (like previous approach)**
   - Rejected: Requires loading entire DB into memory, no selective access

2. **SHA256 for hashing**
   - Rejected: Too fast, vulnerable to GPU attacks, doesn't meet modern security standards

3. **Global salt/pepper instead of per-variable**
   - Rejected: Reduces security, allows correlation attacks

4. **Plaintext salt/pepper**
   - Rejected: Allows offline brute-force if database is stolen

5. **bcrypt/scrypt instead of Argon2id**
   - Rejected: Argon2id is newer and more resistant to attacks

## Test Coverage

- 20 Go unit tests (100% pass)
- 13 Deno E2E tests including:
  - Exhaustive leak detection (API keys, passwords, credit cards, etc.)
  - SQL dump verification (no plaintext in exports)
  - Binary dump verification (no plaintext in raw bytes)
- All tests verify:
  - No plaintext values in database
  - No plaintext names in database
  - No plaintext collections in database
  - Salt and pepper are encrypted
  - Data retrieval works correctly

## References

- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [Argon2 RFC 9106](https://www.rfc-editor.org/rfc/rfc9106.html)
- [Age Encryption Specification](https://age-encryption.org/)
- [SQLite Documentation](https://www.sqlite.org/docs.html)
