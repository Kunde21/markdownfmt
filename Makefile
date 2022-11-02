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
lint: check-gofmt

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
