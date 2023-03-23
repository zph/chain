build:
  goreleaser build --single-target --skip-validate --snapshot --rm-dist

proto:
  buf generate

run cmd:
  go run main.go {{cmd}} chainer

docs:
  go run docs/main.go

tag tagname:
  git tag -a {{tagname}} -m "Release of {{tagname}}"

setup:
  go get golang.org/x/tools/cmd/godoc

test filename:
  ./test/bats/bin/bats {{filename}}

tests:
  ./test/bats/bin/bats test/*.bats
