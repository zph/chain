# CHAIN - secure and convenient temporary storage of credentials

A tool for securely storing and loading secrets into commandline tools.

Inspired by and related to envchain, aws-vault, chamber.

Chain works entirely locally and does not depend on any external services.

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

## Changes
- [x] goreleaser creates binary as `chain`
- [x] setup Github Actions
- [x] setup goreleaser in Github Actions
- [x] Use https://github.com/99designs/keyring with JWT backend
- [x] Remove custom behavior for setting/storing keys and use wrapper tooling
- [x] Use field based logger
- [x] Use age store with expiring keys from initial generation of 10 pub/priv keys
- [x] Setup an `age` based backend to replace JOSE

## TODO
- [ ] Store UUID filename instead of leaking information about what env vars are stored
-   [ ] Use reverse index (EnvToUUID) stored as protobuf in `INDEX` key
-   [ ] Store values as `k/v` pairs with UUID as outer key for filename
- [ ] Setup keyctl with expiring keys
- [ ] Generate docs from commands: https://github.com/spf13/cobra/blob/main/doc/README.md
- [ ] Encrypt .PUBLIC_KEYS to remove threat model of someone tampering with those when re-keying

## Credit

Originally forked from https://github.com/evanphx/schain.
