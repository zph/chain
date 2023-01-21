# 2. Keychain store design

Date: 2023-01-21

## Status

Draft

## Context

Storing keys in keychains is preferable to storing on disk because of gaining security
from the operating system design itself. In this doc we're referring to osx keychain
and linux keyctl keychain.

## Decision

We'll use golang libraries for interfacing with these keychains
- 99designs/keyring for Mac keychain
    - Note: supports keyctl but not the functionality we want and may need CGO (tbd)
- https://github.com/jsipprell/keyctl for Linux keychain (keyctl)
    - Reason: supports key TTLs and does not require CGO

- [ ] Include Expiring keys
    - [ ] No on OSX (system limitation)
    - [ ] Yes on Linux
- [ ] Do we need to individually encrypt each value?
    - [ ] No on OSX
    - [ ] Maybe on Linux
- [ ] Define threat model and security model of keyctl

## Consequences

What becomes easier or more difficult to do and any risks introduced by the change that will need to be mitigated.
