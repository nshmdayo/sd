.PHONY: build test bench lint release clean

build: ## Build the scd binary
	go build -o bin/scd ./cmd/scd

test: ## Run tests
	go test ./... -race -count=1

bench: ## Run benchmarks
	go test ./... -bench=. -benchmem

lint: ## Run static analysis
	golangci-lint run

release: ## Build release with goreleaser
	goreleaser release --clean

clean: ## Remove build artifacts
	rm -rf bin/
