setup_file(){
  just build
  unset $(env | grep CHAIN_ | awk -F= '{print $1}')
}

setup() {
  load 'test_helper/bats-support/load'
  load 'test_helper/bats-assert/load'
  # get the containing directory of this file
  # use $BATS_TEST_FILENAME instead of ${BASH_SOURCE[0]} or $0,
  # as those will point to the bats executable's location or the preprocessed file respectively
  DIR="$( cd "$( dirname "$BATS_TEST_FILENAME" )" >/dev/null 2>&1 && pwd )"
  # make executables in src/ visible to PATH
  PATH="$DIR/../dist/chain_darwin_arm64/:$PATH"
  WORKSPACE="$(mktemp -d -t chain-testing)"
}

teardown_file(){
  echo "$(rm -rf "$WORKSPACE")"
  echo "$(rm -rf $TMPDIR/chain-testing*)"
}

@test "can find our binary" {
  which -a chain
}

@test 'chain help' {
  run chain help
  assert_output <<HERE

     chain is a tool for securely storing and retrieving secrets for use
     in other cli tools.

     For example:
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
     - Designed to use established tools (99designs/keyring) with the File JWT backend (for portability)
     - Requires min password length
     - Offers to generate large secure passwords using "chain password"
     - Never stores env values unencrypted on disk

     Usage:
       chain [command]

     Available Commands:
       completion  Generate the autocompletion script for the specified shell
       create-keys Create keys which will be used with AGE backends
       exec        Execute a command in the context of ENV vars fetched from keychain
       get         Fetch keychain values for <keychain>
       help        Help about any command
       init        Create config file for chain
       password    Generates secure password
       set         Set a key in keychain

     Flags:
       -h, --help   help for chain

     Use "chain [command] --help" for more information about a command.
HERE
}

@test 'chain init carries default values' {
  CHAIN_DIR="$WORKSPACE" CHAIN_STORE=1 CHAIN_LOG_LEVEL=info chain init
  run cat $WORKSPACE/.chain.hcl
  assert_output --partial - <<STRING
"keyring_user" = ""

"log_level" = "info"

"password" = ""

"password_validation_length" = 20

"store" = "1"
STRING

  assert_output --partial '"password" = ""'
  assert_output --regexp '"dir" = ".*chain-testing.*"'
}

@test 'chain init carries non-default values' {
  CHAIN_DIR="$WORKSPACE" CHAIN_LOG_LEVEL=info chain init
  run cat $WORKSPACE/.chain.hcl
  assert_output --partial - <<STRING
"keyring_user" = ""

"log_level" = "info"

"password" = ""

"password_validation_length" = 20

"store" = "4"
STRING

  assert_output --partial '"password" = ""'
  assert_output --regexp '"dir" = ".*chain-testing.*"'
}

