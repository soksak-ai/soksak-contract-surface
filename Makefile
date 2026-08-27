SHELL := /bin/sh

.PHONY: preflight prepare build verify

preflight:
	@scripts/check-build-environment.sh

prepare: preflight
	@go mod download
	@go mod verify
	@PATH="$$HOME/.cargo/bin:$$PATH" cargo fetch --locked

build: prepare
	@go build ./...
	@PATH="$$HOME/.cargo/bin:$$PATH" cargo build --locked --release

verify: prepare
	@go mod tidy -diff
	@go test -count=1 ./...
	@go vet ./...
	@PATH="$$HOME/.cargo/bin:$$PATH" cargo test --locked --release
