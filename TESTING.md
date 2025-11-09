# Testing Harness Plan for Chain CLI

## Architecture Overview

**Two-Tier Testing Strategy:**
1. **E2E Tests (Deno/TypeScript)** - Test CLI behavior from user perspective
2. **Unit Tests (Go)** - Verify encryption, storage integrity, and internal logic

---

## E2E Test Harness (Deno + TypeScript)

### Directory Structure
```
test/
├── chain.test.ts           # Main e2e test suite
├── helpers.ts              # Shared test utilities
├── fixtures/               # Test data and sample configs
│   └── test-vars.env
└── deno.json              # Deno configuration
```

### Core Helper Functions

**1. Chain CLI Executor**
```typescript
interface ChainResult {
  code: number
  out: string
  lines: string[]
  asKeyVal: [string, string][]
  store: string
}

const chain = async (
  command: string,
  env: Record<string, string> = {}
): Promise<ChainResult> => {
  const result = await $.raw`./chain ${command}`
    .env({ ...env, CHAIN_DIR: testDir })
    .stderr("inherit")
    .stdout("piped")
    .captureCombined()
    .noThrow()

  return {
    code: result.code,
    out: result.stdout.trimEnd(),
    lines: result.stdout.trimEnd().split("\n"),
    asKeyVal: result.stdout.trimEnd()
      .split("\n")
      .map(l => l.split("=") as [string, string]),
    store: env.CHAIN_STORE || "1"
  }
}
```

**2. Test Isolation Helpers**
```typescript
const setupTestEnv = () => {
  const testDir = Deno.makeTempDirSync({ prefix: "chain-test-" })
  return { CHAIN_DIR: testDir }
}

const cleanupTestEnv = (env: { CHAIN_DIR: string }) => {
  Deno.removeSync(env.CHAIN_DIR, { recursive: true })
}
```

**3. Encryption Verification Helper**
```typescript
const verifyFileEncrypted = async (filePath: string): Promise<boolean> => {
  const content = await Deno.readFile(filePath)
  // Check for age encryption header
  const header = new TextDecoder().decode(content.slice(0, 100))
  return header.includes("age-encryption.org") ||
         !isPlaintext(content)
}

const isPlaintext = (data: Uint8Array): boolean => {
  // Heuristic: check if majority of bytes are printable ASCII
  let printable = 0
  for (let i = 0; i < Math.min(data.length, 100); i++) {
    if (data[i] >= 32 && data[i] <= 126) printable++
  }
  return printable / Math.min(data.length, 100) > 0.9
}
```

---

## E2E Test Scenarios

### 1. **Basic CRUD Operations**
```typescript
Deno.test("set and get variables", async () => {
  const env = setupTestEnv()

  // Set variables
  const setResult = await chain(
    `set test-chain`,
    { ...env, CHAIN_STORE: "6", CHAIN_PASSWORD: testPassword }
  ).stdin("FOO=bar\nBAZ=qux\n")

  assertEquals(setResult.code, 0)
  assertStringIncludes(setResult.out, "Value(s) saved: 2")

  // Get variables
  const getResult = await chain(
    `get test-chain`,
    { ...env, CHAIN_STORE: "6", CHAIN_PASSWORD: testPassword }
  )

  assertEquals(getResult.code, 0)
  assertEquals(getResult.asKeyVal, [
    ["BAZ", "qux"],
    ["FOO", "bar"]
  ])

  cleanupTestEnv(env)
})
```

### 2. **Multi-Storage Backend Tests**
```typescript
for (const [storeType, storeName] of [
  ["1", "StandardStore"],
  ["3", "KeychainByPlatform"],
  ["4", "AgeStore"],
  ["6", "SQLiteStore"]
]) {
  Deno.test(`${storeName}: set/get workflow`, async () => {
    // Test implementation
  })
}
```

### 3. **Exec Command Integration**
```typescript
Deno.test("exec loads variables into environment", async () => {
  const env = setupTestEnv()

  await chain(`set test-chain`, env)
    .stdin("TEST_VAR=hello\n")

  const execResult = await chain(
    `exec test-chain -- env`,
    env
  )

  assertStringIncludes(execResult.out, "TEST_VAR=hello")
  cleanupTestEnv(env)
})
```

### 4. **Encryption Verification**
```typescript
Deno.test("SQLite store encrypts database file", async () => {
  const env = setupTestEnv()

  await chain(`set test-chain`, {
    ...env,
    CHAIN_STORE: "6",
    CHAIN_PASSWORD: testPassword
  }).stdin("SECRET=password123\n")

  const dbPath = `${env.CHAIN_DIR}/test-chain/chain.db.age`
  const encrypted = await verifyFileEncrypted(dbPath)

  assert(encrypted, "Database file should be encrypted")

  // Verify secret is NOT in plaintext
  const raw = await Deno.readTextFile(dbPath)
  assertFalse(raw.includes("password123"))
  assertFalse(raw.includes("SECRET"))

  cleanupTestEnv(env)
})
```

### 5. **Age Store Key Management**
```typescript
Deno.test("create-keys generates valid age keys", async () => {
  const env = setupTestEnv()

  const result = await chain(
    `create-keys test-chain 3`,
    env
  )

  assertEquals(result.code, 0)
  assertEquals(result.lines.length, 4) // Header + 3 keys

  // Verify .PUBLIC_KEYS file created
  const pubKeysPath = `${env.CHAIN_DIR}/test-chain/.PUBLIC_KEYS`
  assert(await exists(pubKeysPath))

  cleanupTestEnv(env)
})
```

### 6. **Error Handling**
```typescript
Deno.test("get non-existent chain returns error", async () => {
  const env = setupTestEnv()

  const result = await chain(`get non-existent`, env)

  assertNotEquals(result.code, 0)

  cleanupTestEnv(env)
})
```

---

## Unit Tests (Go)

### Directory Structure
```
cmd/
├── sqlite_store_test.go
├── age_store_test.go
├── stores_test.go
└── lib_test.go
```

### 1. **SQLite Encryption Tests** (`cmd/sqlite_store_test.go`)
```go
func TestSQLiteStore_EncryptsDatabase(t *testing.T) {
    // Setup
    tmpDir := t.TempDir()
    viper.Set(ChainDirKey, tmpDir)
    viper.Set(KeyringPassword, testPassword)

    store, err := NewSQLiteStore("test-chain")
    require.NoError(t, err)
    defer store.PostRunHook()

    // Set a value
    err = store.Set(keyring.Item{
        Key: "SECRET_KEY",
        Data: []byte("super-secret-value"),
    })
    require.NoError(t, err)

    // Force save to disk
    err = store.saveToDisk()
    require.NoError(t, err)

    // Verify file is encrypted
    dbPath := filepath.Join(tmpDir, "test-chain", "chain.db.age")
    content, err := os.ReadFile(dbPath)
    require.NoError(t, err)

    // Should not contain plaintext secret
    assert.NotContains(t, string(content), "super-secret-value")
    assert.NotContains(t, string(content), "SECRET_KEY")

    // Should have age encryption marker
    assert.True(t, bytes.Contains(content, []byte("age-encryption.org")) ||
                !isPrintableASCII(content))
}

func TestSQLiteStore_DecryptsAndLoadsInMemory(t *testing.T) {
    // Setup and save encrypted DB
    tmpDir := t.TempDir()
    // ... create and save ...

    // Create new store instance (simulates restart)
    store2, err := NewSQLiteStore("test-chain")
    require.NoError(t, err)

    // Verify data loaded from encrypted file
    item, err := store2.Get("SECRET_KEY")
    require.NoError(t, err)
    assert.Equal(t, "super-secret-value", string(item.Data))
}

func TestSQLiteStore_SavesOnPostRunHook(t *testing.T) {
    // Verify PostRunHook saves in-memory DB to encrypted file
}
```

### 2. **Age Store Encryption Tests** (`cmd/age_store_test.go`)
```go
func TestAgeStore_EncryptsIndividualFiles(t *testing.T) {
    // Verify each secret stored as encrypted file
}

func TestAgeStore_RequiresValidKey(t *testing.T) {
    // Verify decryption fails with wrong key
}
```

### 3. **Store Interface Tests** (`cmd/stores_test.go`)
```go
func TestAllStores_ImplementInterface(t *testing.T) {
    stores := []struct{
        name string
        factory func(string) (Store, error)
    }{
        {"SQLite", NewSQLiteStore},
        {"Age", NewAgeStore},
        {"Standard", NewStandardStore},
        // ...
    }

    for _, tc := range stores {
        t.Run(tc.name, func(t *testing.T) {
            store, err := tc.factory("test")
            require.NoError(t, err)

            // Verify interface methods work
            testStoreInterface(t, store)
        })
    }
}
```

---

## Test Configuration Files

### `test/deno.json`
```json
{
  "tasks": {
    "test": "deno test --allow-all test/",
    "test:watch": "deno test --allow-all --watch test/"
  },
  "imports": {
    "@david/dax": "jsr:@david/dax",
    "@std/assert": "jsr:@std/assert"
  }
}
```

### `Justfile` additions
```just
test-e2e:
  go build -o chain .
  deno test --allow-all test/

test-unit:
  go test -v ./cmd/...

test: test-unit test-e2e

test-coverage:
  go test -coverprofile=coverage.out ./cmd/...
  go tool cover -html=coverage.out
```

---

## Implementation Checklist

**Phase 1: Unit Tests**
- [ ] Add Go test files for each store
- [ ] Implement encryption verification tests
- [ ] Test store interface compliance
- [ ] Add table-driven tests for all storage backends

**Phase 2: E2E Harness**
- [ ] Create test/ directory structure
- [ ] Implement helper functions
- [ ] Add deno.json configuration
- [ ] Create fixture data

**Phase 3: E2E Test Suite**
- [ ] Basic CRUD tests
- [ ] Multi-backend tests
- [ ] Encryption verification tests
- [ ] Error handling tests
- [ ] Integration tests (exec, create-keys)

**Phase 4: CI Integration**
- [ ] Add GitHub Actions workflow
- [ ] Run both unit and e2e tests
- [ ] Generate coverage reports
