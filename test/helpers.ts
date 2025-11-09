import $ from "@david/dax";

export interface ChainResult {
  code: number;
  out: string;
  lines: string[];
  asKeyVal: [string, string][];
  store: string;
}

export const chain = async (
  command: string,
  env: Record<string, string> = {},
  stdin?: string
): Promise<ChainResult> => {
  // Use the shared binary built before tests
  const chainPath = "go run main.go";

  let cmd = $.raw`${chainPath} ${command}`
    .env(env)
    .stderr("piped")
    .stdout("piped")
    .noThrow();

  if (stdin) {
    cmd = cmd.stdinText(stdin);
  }

  const result = await cmd;

  return {
    code: result.code,
    out: result.stdout.trimEnd(),
    lines: result.stdout.trimEnd().split("\n").filter(l => l.length > 0),
    asKeyVal: result.stdout
      .trimEnd()
      .split("\n")
      .filter(l => l.includes("="))
      .map((l) => {
        const [key, ...rest] = l.split("=");
        return [key, rest.join("=")] as [string, string];
      }),
    store: env.CHAIN_STORE || "1",
  };
};

export const setupTestEnv = (): { CHAIN_DIR: string } => {
  const testDir = Deno.makeTempDirSync({ prefix: "chain-test-" });
  return { CHAIN_DIR: testDir };
};

export const cleanupTestEnv = (env: { CHAIN_DIR: string }) => {
  try {
    Deno.removeSync(env.CHAIN_DIR, { recursive: true });
  } catch (e) {
    // Ignore cleanup errors
  }
};

export const verifyFileEncrypted = async (
  filePath: string
): Promise<boolean> => {
  try {
    const content = await Deno.readFile(filePath);
    // Check for age encryption header
    const header = new TextDecoder().decode(content.slice(0, 100));
    return (
      header.includes("age-encryption.org") || !isPlaintext(content)
    );
  } catch {
    return false;
  }
};

export const isPlaintext = (data: Uint8Array): boolean => {
  // Heuristic: check if majority of bytes are printable ASCII
  let printable = 0;
  const checkLength = Math.min(data.length, 200);

  for (let i = 0; i < checkLength; i++) {
    if ((data[i] >= 32 && data[i] <= 126) || data[i] === 10 || data[i] === 13) {
      printable++;
    }
  }

  return printable / checkLength > 0.8;
};

export const fileContains = async (
  filePath: string,
  searchString: string
): Promise<boolean> => {
  try {
    const content = await Deno.readTextFile(filePath);
    return content.includes(searchString);
  } catch {
    try {
      const content = await Deno.readFile(filePath);
      const text = new TextDecoder().decode(content);
      return text.includes(searchString);
    } catch {
      return false;
    }
  }
};

export const exists = async (filePath: string): Promise<boolean> => {
  try {
    await Deno.stat(filePath);
    return true;
  } catch {
    return false;
  }
};

export const decryptAndDumpSQLite = async (
  dbPath: string,
  ageKey: string
): Promise<string> => {
  // Extract private key from format: publicPrefix:privateKey
  const [, privateKey] = ageKey.split(":");
  if (!privateKey) {
    throw new Error("Invalid age key format");
  }

  // Create temp file for age identity
  const identityFile = await Deno.makeTempFile({ prefix: "age-identity-" });
  try {
    await Deno.writeTextFile(identityFile, privateKey);

    // Decrypt using age CLI
    const result = await $`age --decrypt -i ${identityFile} ${dbPath}`
      .stdout("piped")
      .stderr("piped")
      .noThrow();

    if (result.code !== 0) {
      throw new Error(`Failed to decrypt: ${result.stderr}`);
    }

    return result.stdout;
  } finally {
    try {
      await Deno.remove(identityFile);
    } catch {
      // Ignore cleanup errors
    }
  }
};

export const readFileAsText = async (filePath: string): Promise<string> => {
  try {
    return await Deno.readTextFile(filePath);
  } catch {
    const content = await Deno.readFile(filePath);
    return new TextDecoder("utf-8", { fatal: false }).decode(content);
  }
};
