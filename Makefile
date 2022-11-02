# All .go files checked into the repository
# must be properly gofmt-ed.
GOFMT_FILES = $(shell git ls-files '*.go')

.PHONY: build
build: ## Builds all packages.
	go build -v ./...

.PHONY: test
test: ## Tests all packages.
	go test -v ./...

.PHONY: lint
lint: ## Runs various analyses on the code
lint: check-gofmt check-tidy

.PHONY: gofmt
gofmt: ## Makes all files gofmt compliant.
	gofmt -w $(GOFMT_FILES)

.PHONY: check-gofmt
check-gofmt: ## Checks that all files are gofmt-compliant.
	@DIFF=$$(gofmt -d $(GOFMT_FILES)); \
	if [ -n "$$DIFF" ]; then \
		echo "--- gofmt would change:"; \
		echo "$$DIFF"; \
		echo "Run 'make gofmt' to fix"; \
		exit 1; \
	fi

.PHONY: tidy
tidy: ## Makes go.mod and go.sum up-to-date
	go mod tidy -v

.PHONY: check-tidy
check-tidy: ## Checks that go.mod/go.sum are up-to-date.
check-tidy: tidy
	@DIFF=$$(git diff go.mod go.sum); \
	if [ -n "$$DIFF" ]; then \
		echo "--- go.mod/go.sum are out of date:"; \
		echo "$$DIFF"; \
		echo "Run 'make tidy' to fix"; \
		exit 1; \
	fi
