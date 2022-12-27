# CHAIN - secure and convenient temporary storage of credentials

A tool for securely storing and loading secrets into commandline tools.

Inspired by and related to envchain, aws-vault, chamber.

Chain works entirely locally and does not depend on any external services.

## Usage
```
echo "AWS_SECRET_KEY_ID=FAKEKEY" | chain set aws-creds
chain get aws-creds
chain exec aws-creds -- aws s3 ls...

# Store your one or more env variables
chain set chain-name<ENTER>

# Fetch to review them
chain get chain-name

# Execute a secondary command in the environment of these variables
chain exec chain-name -- aws s3 ...

# ENV variables
CHAIN_PASSWORD=<password used in keychain for storing key>
CHAIN_DIR=<directory for files stored on disk, default=.chain>

# Values can be set in a .chain.hcl configuration file
Use "chain init" to create the init file in .chain/.chain.hcl

Security:
- Designed to use established tools (99designs/keyring) with the File encrypted JWT backend (for portability)
- Requires min password length
- Offers to generate large secure passwords using "chain password"
- Never stores env values unencrypted on disk

Usage:
  chain [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  exec        Execute a command in the context of ENV vars fetched from keychain
  get         Fetch keychain values for <keychain>
  help        Help about any command
  init        Create config file for chain
  password    Generates secure password
  set         Set a key in keychain

Flags:
  -h, --help   help for chain

Use "chain [command] --help" for more information about a command.
```

## Changes
- [x] goreleaser creates binary as `chain`
- [x] setup Github Actions
- [x] setup goreleaser in Github Actions
- [x] Use https://github.com/99designs/keyring with JWT backend
- [x] Remove custom behavior for setting/storing keys and use wrapper tooling

## TODO
- [ ] Setup an `age` based backend to replace JOSE
- [ ] Store UUID filename instead of leaking information about what env vars are stored
-   [ ] Use reverse index (EnvToUUID) stored as protobuf in `INDEX` key
-   [ ] Store values as `k/v` pairs with UUID as outer key for filename
- [ ] Use field based logger
- [ ] Setup keyctl with expiring keys

## Credit

Originally forked from https://github.com/evanphx/schain.
