# CHAIN - secure and convenient temporary storage of credentials

A tool for securely storing and loading secrets into commandline tools.

Inspired by and related to envchain, aws-vault, chamber.

Chain works entirely locally and does not depend on any external services.

## Installation

Methods:

1. Download from [Releases](https://github.com/zph/chain/releases) and place on $PATH
2. Use hermit with custom source
   1. Setup hermit for project level tooling: https://cashapp.github.io/hermit/usage/get-started/
   2. Source: https://github.com/zph/hermit-packages
   3. `hermit install chain`

## Usage

See [docs](./docs/chain.md) for full commands

```
echo "AWS_SECRET_KEY_ID=FAKEKEY" | chain set aws-creds
chain get aws-creds
chain exec aws-creds -- aws s3 ls...

# ENV variables
CHAIN_PASSWORD=<password used in keychain for storing key>
CHAIN_STORE=[1-5 see chain.proto for examples]
CHAIN_DIR=<directory for files stored on disk, default=.chain>
```

See the [proto](chain/v1/chain.proto) for which stores are available and their respective `cmd/*_store.go` and [stores](cmd/stores.go) files for implementation. They can also be seen in [proto](chain/v1/chain.proto).
## Changes
- [x] goreleaser creates binary as `chain`
- [x] setup Github Actions
- [x] setup goreleaser in Github Actions
- [x] Use https://github.com/99designs/keyring with JWT backend
- [x] Remove custom behavior for setting/storing keys and use wrapper tooling
- [x] Use field based logger
- [x] Use age store with expiring keys from initial generation of 10 pub/priv keys
- [x] Setup an `age` based backend to replace JOSE
- [x] Generate docs from commands: https://github.com/spf13/cobra/blob/main/doc/README.md

## TODO
- [ ] Store UUID filename instead of leaking information about what env vars are stored
-   [ ] Use reverse index (EnvToUUID) stored as protobuf in `INDEX` key
-   [ ] Store values as `k/v` pairs with UUID as outer key for filename
- [ ] Setup backend for keychain that either uses osx-keychain OR keyctcl
    - [ ] Setup keyctl with expiring keys (https://github.com/jsipprell/keyctl)
    - [ ] See design notes
- [ ] Encrypt .PUBLIC_KEYS to remove threat model of someone tampering with those when re-keying
- [ ] Include chain positional arg as part of config and switch to being a global flag rather than positional
arg. Allows for better common ergonomics.
- [ ] Setup bats testing

## Credit

Originally forked from https://github.com/evanphx/schain.
