
## test-all: Running both unit and swamp tests
test:
	@echo "--> Running all tests without data race detector"
	@go test ./...
	@echo "--> Running all tests with data race detector"
	@go test -race ./...
.PHONY: test

PB_PKGS=$(shell find . -name 'pb' -type d)
PB_CORE=$(shell go list -f {{.Dir}} -m github.com/tendermint/tendermint)
PB_GOGO=$(shell go list -f {{.Dir}} -m github.com/gogo/protobuf)

## pb-gen: Generate protobuf code for all /pb/*.proto files in the project.
pb-gen:
	@echo '--> Generating protobuf'
	@for dir in $(PB_PKGS); \
		do for file in `find $$dir -type f -name "*.proto"`; \
			do protoc -I=. -I=${PB_CORE}/proto/ -I=${PB_GOGO}  --gogofaster_out=paths=source_relative:. $$file; \
			echo '-->' $$file; \
		done; \
	done;
.PHONY: pb-gen
