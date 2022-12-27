build:
  goreleaser build --single-target --skip-validate --snapshot --rm-dist

proto:
  buf generate

run cmd:
  go run main.go {{cmd}} chainer
