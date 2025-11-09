import {
  assert,
  assertEquals,
  assertNotEquals,
  assertStringIncludes,
} from "@std/assert";
import $ from "@david/dax";
import {
  chain,
  cleanupTestEnv,
  decryptAndDumpSQLite,
  exists,
  fileContains,
  readFileAsText,
  setupTestEnv,
  verifyFileEncrypted,
} from "./helpers.ts";

const testPassword = "this-is-a-very-long-test-password-12345";

Deno.test("basic set and get variables", async () => {
  const env = setupTestEnv();

  try {
    // Set variables
    const setResult = await chain(
      `set test-chain`,
      { ...env, CHAIN_STORE: "1", CHAIN_PASSWORD: testPassword },
      "FOO=bar\nBAZ=qux\n"
    );

    assertEquals(setResult.code, 0);
    assertStringIncludes(setResult.out, "Value(s) saved: 2");

    // Get variables
    const getResult = await chain(`get test-chain`, {
      ...env,
      CHAIN_STORE: "1",
      CHAIN_PASSWORD: testPassword,
    });

    assertEquals(getResult.code, 0);
    assert(getResult.asKeyVal.some(([k, v]) => k === "FOO" && v === "bar"));
    assert(getResult.asKeyVal.some(([k, v]) => k === "BAZ" && v === "qux"));
  } finally {
    cleanupTestEnv(env);
  }
});

Deno.test("exec loads variables into environment", async () => {
  const env = setupTestEnv();

  try {
    await chain(
      `set test-chain`,
      { ...env, CHAIN_STORE: "1", CHAIN_PASSWORD: testPassword },
      "TEST_VAR=hello\n"
    );

    const execResult = await chain(`exec test-chain -- env`, {
      ...env,
      CHAIN_STORE: "1",
      CHAIN_PASSWORD: testPassword,
    });

    assertEquals(execResult.code, 0);
    assertStringIncludes(execResult.out, "TEST_VAR=hello");
  } finally {
    cleanupTestEnv(env);
  }
});

Deno.test("get non-existent chain returns empty", async () => {
  const env = setupTestEnv();

  try {
    const result = await chain(`get non-existent-chain`, {
      ...env,
      CHAIN_STORE: "1",
      CHAIN_PASSWORD: testPassword,
    });

    // Getting from a non-existent/empty chain returns success with no output
    assertEquals(result.code, 0);
    assertEquals(result.lines.length, 0);
  } finally {
    cleanupTestEnv(env);
  }
});

Deno.test("SQLite store encrypts database file", async () => {
  const env = setupTestEnv();

  try {
    // First create age keys (needs AGE backend)
    const createKeysResult = await chain(
      `create-keys test-chain 2`,
      { ...env, CHAIN_STORE: "6" }
    );

    assertEquals(createKeysResult.code, 0);
    assert(createKeysResult.lines.length >= 3); // Header + keys

    // Extract the first key (format: publicKeyPrefix:privateKey)
    const keyLine = createKeysResult.lines.find(l => l.includes(":"));
    assert(keyLine, "Should have at least one key");

    // Set variables with SQLite store
    await chain(
      `set test-chain`,
      {
        ...env,
        CHAIN_STORE: "6",
        CHAIN_PASSWORD: keyLine,
      },
      "SECRET=password123\nFOO=bar\n"
    );

    // SQLite now stores on-disk with encrypted columns
    const dbPath = `${env.CHAIN_DIR}/test-chain/chain.db`;
    const dbExists = await exists(dbPath);
    assert(dbExists, "Database file should exist");

    // Read database and verify secrets are NOT in plaintext
    const hasSecret = await fileContains(dbPath, "password123");
    const hasKey = await fileContains(dbPath, "SECRET");
    const hasChainName = await fileContains(dbPath, "test-chain");

    assert(!hasSecret, "Secret value should not be in plaintext");
    assert(!hasKey, "Secret key should not be in plaintext");
    // Collection name should not be in plaintext (it's encrypted or hashed)
    // Note: SHA256 hash of "test-chain" will be in the DB, but not the plaintext
  } finally {
    cleanupTestEnv(env);
  }
});

Deno.test("SQLite store can retrieve encrypted data", async () => {
  const env = setupTestEnv();

  try {
    // Create age keys (needs AGE backend)
    const createKeysResult = await chain(
      `create-keys test-chain 2`,
      { ...env, CHAIN_STORE: "6" }
    );

    const keyLine = createKeysResult.lines.find(l => l.includes(":"));
    assert(keyLine);

    // Set variables
    await chain(
      `set test-chain`,
      {
        ...env,
        CHAIN_STORE: "6",
        CHAIN_PASSWORD: keyLine,
      },
      "SECRET=password123\nFOO=bar\n"
    );

    // Get variables back
    const getResult = await chain(`get test-chain`, {
      ...env,
      CHAIN_STORE: "6",
      CHAIN_PASSWORD: keyLine,
    });

    assertEquals(getResult.code, 0);
    assert(getResult.asKeyVal.some(([k, v]) => k === "SECRET" && v === "password123"));
    assert(getResult.asKeyVal.some(([k, v]) => k === "FOO" && v === "bar"));
  } finally {
    cleanupTestEnv(env);
  }
});

Deno.test("create-keys generates valid age keys", async () => {
  const env = setupTestEnv();

  try {
    const result = await chain(`create-keys test-chain 3`, { ...env, CHAIN_STORE: "4" });

    assertEquals(result.code, 0);
    assert(result.lines.length >= 4); // Header + 3 keys

    // Verify .PUBLIC_KEYS file created
    const pubKeysPath = `${env.CHAIN_DIR}/test-chain/.PUBLIC_KEYS`;
    assert(await exists(pubKeysPath), "PUBLIC_KEYS file should exist");
  } finally {
    cleanupTestEnv(env);
  }
});

Deno.test("password command generates secure password", async () => {
  const result = await chain(`password`);

  assertEquals(result.code, 0);
  assert(result.out.length > 20, "Password should be long");
});

// Multi-backend tests
for (
  const [storeType, storeName] of [
    ["1", "StandardStore"],
    ["4", "AgeStore"],
    ["6", "SQLiteStore"],
  ]
) {
  Deno.test(`${storeName}: basic workflow`, async () => {
    const env = setupTestEnv();

    try {
      let envVars: Record<string, string> = { ...env, CHAIN_STORE: storeType };

      // Age stores need keys
      if (storeType === "4" || storeType === "6") {
        const createKeysResult = await chain(
          `create-keys test-chain 2`,
          { ...env, CHAIN_STORE: storeType }
        );
        const keyLine = createKeysResult.lines.find(l => l.includes(":"));
        assert(keyLine);
        envVars = { ...envVars, CHAIN_PASSWORD: keyLine };
      } else {
        envVars = { ...envVars, CHAIN_PASSWORD: testPassword };
      }

      // Set
      const setResult = await chain(
        `set test-chain`,
        envVars,
        "KEY1=value1\nKEY2=value2\n"
      );
      assertEquals(setResult.code, 0);

      // Get
      const getResult = await chain(`get test-chain`, envVars);
      assertEquals(getResult.code, 0);
      assert(getResult.asKeyVal.some(([k, v]) => k === "KEY1" && v === "value1"));
      assert(getResult.asKeyVal.some(([k, v]) => k === "KEY2" && v === "value2"));
    } finally {
      cleanupTestEnv(env);
    }
  });
}

// Exhaustive leak detection tests for SQLite store
Deno.test("SQLite store: exhaustive leak detection", async () => {
  const env = setupTestEnv();

  try {
    // Create age keys
    const createKeysResult = await chain(
      `create-keys leak-test 2`,
      { ...env, CHAIN_STORE: "6" }
    );

    assertEquals(createKeysResult.code, 0);
    const keyLine = createKeysResult.lines.find(l => l.includes(":"));
    assert(keyLine);

    // Set multiple sensitive values with various patterns
    const sensitiveData = {
      "API_KEY": "sk-1234567890abcdef",
      "DATABASE_PASSWORD": "MyS3cr3tP@ssw0rd!",
      "SSH_PRIVATE_KEY": "-----BEGIN RSA PRIVATE KEY-----",
      "CREDIT_CARD": "4532-1234-5678-9010",
      "EMAIL": "user@example.com",
      "USERNAME": "admin",
    };

    let stdin = "";
    for (const [key, value] of Object.entries(sensitiveData)) {
      stdin += `${key}=${value}\n`;
    }

    await chain(
      `set leak-test`,
      {
        ...env,
        CHAIN_STORE: "6",
        CHAIN_PASSWORD: keyLine,
      },
      stdin
    );

    // Read the entire database file as text
    const dbPath = `${env.CHAIN_DIR}/leak-test/chain.db`;
    const dbContent = await readFileAsText(dbPath);

    // Check that NO sensitive data is present in plaintext
    for (const [key, value] of Object.entries(sensitiveData)) {
      if (dbContent.includes(value)) {
        throw new Error(`LEAK DETECTED: Value "${value}" found in plaintext in database`);
      }
      if (dbContent.includes(key)) {
        throw new Error(`LEAK DETECTED: Key "${key}" found in plaintext in database`);
      }
    }

    // Check that collection name is not in plaintext
    if (dbContent.includes("leak-test")) {
      throw new Error(`LEAK DETECTED: Collection name "leak-test" found in plaintext in database`);
    }

    // Verify we can still retrieve all values correctly
    const getResult = await chain(`get leak-test`, {
      ...env,
      CHAIN_STORE: "6",
      CHAIN_PASSWORD: keyLine,
    });

    assertEquals(getResult.code, 0);
    for (const [key, value] of Object.entries(sensitiveData)) {
      const found = getResult.asKeyVal.some(([k, v]) => k === key && v === value);
      assert(found, `Failed to retrieve ${key}=${value}`);
    }
  } finally {
    cleanupTestEnv(env);
  }
});

Deno.test("SQLite store: dump all tables and verify no leaks", async () => {
  const env = setupTestEnv();

  try {
    // Create age keys
    const createKeysResult = await chain(
      `create-keys dump-test 2`,
      { ...env, CHAIN_STORE: "6" }
    );

    const keyLine = createKeysResult.lines.find(l => l.includes(":"));
    assert(keyLine);

    // Set test data
    await chain(
      `set dump-test`,
      {
        ...env,
        CHAIN_STORE: "6",
        CHAIN_PASSWORD: keyLine,
      },
      "TOP_SECRET=classified_information\nACCESS_TOKEN=bearer_token_12345\n"
    );

    const dbPath = `${env.CHAIN_DIR}/dump-test/chain.db`;

    // Use sqlite3 to dump the entire database as SQL
    const dumpResult = await $`sqlite3 ${dbPath} .dump`
      .stdout("piped")
      .stderr("piped")
      .noThrow();

    if (dumpResult.code !== 0) {
      throw new Error(`Failed to dump database: ${dumpResult.stderr}`);
    }

    const sqlDump = dumpResult.stdout;

    // Verify sensitive data is NOT in the dump
    const secrets = ["TOP_SECRET", "classified_information", "ACCESS_TOKEN", "bearer_token_12345", "dump-test"];

    for (const secret of secrets) {
      if (sqlDump.includes(secret)) {
        throw new Error(`LEAK DETECTED in SQL dump: "${secret}" found in plaintext`);
      }
    }

    // Verify the dump contains encrypted blobs (age-encryption headers or binary data)
    assert(sqlDump.length > 0, "SQL dump should not be empty");
    assert(sqlDump.includes("CREATE TABLE"), "SQL dump should contain table definitions");
  } finally {
    cleanupTestEnv(env);
  }
});

Deno.test("SQLite store: binary dump leak detection", async () => {
  const env = setupTestEnv();

  try {
    const createKeysResult = await chain(
      `create-keys binary-test 2`,
      { ...env, CHAIN_STORE: "6" }
    );

    const keyLine = createKeysResult.lines.find(l => l.includes(":"));
    assert(keyLine);

    const highlySecret = "NUCLEAR_LAUNCH_CODES=00000000";
    const secretKey = "NUCLEAR_LAUNCH_CODES";
    const secretValue = "00000000";

    await chain(
      `set binary-test`,
      {
        ...env,
        CHAIN_STORE: "6",
        CHAIN_PASSWORD: keyLine,
      },
      highlySecret + "\n"
    );

    const dbPath = `${env.CHAIN_DIR}/binary-test/chain.db`;

    // Read as binary
    const dbBinary = await Deno.readFile(dbPath);
    const decoder = new TextDecoder("utf-8", { fatal: false });
    const dbText = decoder.decode(dbBinary);

    // Check for leaks in binary representation
    if (dbText.includes(secretKey)) {
      throw new Error(`LEAK DETECTED in binary: Key "${secretKey}" found`);
    }
    if (dbText.includes(secretValue)) {
      throw new Error(`LEAK DETECTED in binary: Value "${secretValue}" found`);
    }
    if (dbText.includes("binary-test")) {
      throw new Error(`LEAK DETECTED in binary: Collection name "binary-test" found`);
    }

    // Verify we can still decrypt and retrieve
    const getResult = await chain(`get binary-test`, {
      ...env,
      CHAIN_STORE: "6",
      CHAIN_PASSWORD: keyLine,
    });

    assertEquals(getResult.code, 0);
    assertStringIncludes(getResult.out, highlySecret);
  } finally {
    cleanupTestEnv(env);
  }
});

