BIN=acos

.PHONY: help
## help: prints this help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: build
## build: build the application
build: clean
	@echo "Building..."
	@cargo build --release
	@mkdir -p ./dist
	@cp ./target/release/${BIN} ./dist/${BIN}

.PHONY: run
## run: runs the application
run:
	cargo run

.PHONY: clean
## clean: cleans the binary
clean:
	@echo "Cleaning"
	@cargo clean
	@rm -rf ./dist

.PHONY: test
## test: runs tests
test:
	cargo test

.PHONY: fmt
## fmt: formats the code
fmt:
	cargo fmt

.PHONY: lint
## lint: runs clippy linter
lint:
	cargo clippy -- -D warnings
