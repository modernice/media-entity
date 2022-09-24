.PHONY: generate
generate:
	@./scripts/generate

.PHONY: test
test:
	@go test -v ./...
	@go test -v ./api/proto/...
	@go test -v ./goes/esgallery/...
