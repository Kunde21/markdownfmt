.PHONY: build
build: ## Builds all packages.
	go build -v ./...

.PHONY: test
test: ## Tests all packages.
	go test -v ./...

.PHONY: lint
lint: ## Runs various analyses on the code
