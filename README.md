# CHAIN - envchain in golang

A tool for securely storing and loading secrets into commandline tools.

Inspired by and related to envchain, aws-vault, chamber.

Chain works entirely locally and does not depend on any external services.

## Changes
[x] - goreleaser creates binary as `chain`
[x] - setup Github Actions
[x] - setup goreleaser in Github Actions
[x] - Use https://github.com/99designs/keyring with JWT backend
[x] - Remove custom behavior for setting/storing keys and use wrapper tooling

## TODO
[ ] - Setup an `age` based backend to replace JOSE
[ ] - Store UUID filename instead of leaking information about what env vars are stored
  [ ] - Use reverse index (EnvToUUID) stored as protobuf in `INDEX` key
  [ ] - Store values as `k/v` pairs with UUID as outer key for filename
[ ] - Use field based logger
[ ] - Setup keyctl with expiring keys

## Credit

Originally forked from https://github.com/evanphx/schain.
