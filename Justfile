build:
  goreleaser build --single-target --skip-validate --snapshot --rm-dist

proto:
  buf generate
