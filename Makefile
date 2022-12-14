# All .go files checked into the repository
# must be properly gofmt-ed.
GOFMT_FILES = $(shell git ls-files '*.go')

# Non-test source files.
SRC_FILES = $(shell git ls-files '*.go' | grep -v '_test.go$$')

# List of Markdown files that are required to be markdownfmt-compliant.
MDFMT_FILES = README.md CHANGELOG.md

MARKDOWNFMT = bin/markdownfmt

.PHONY: help
help: ## Prints list of targets and help for them.
	@grep -F '##' $(MAKEFILE_LIST) | grep -v grep | sed -e 's/##/\t/' | \
		column -t -s $$'\t'

.PHONY: build
build: ## Builds all packages.
	go build -v ./...

.PHONY: test
test: ## Tests all packages.
	go test -v ./...

.PHONY: lint
lint: ## Runs various analyses on the code.
lint: check-gofmt check-tidy check-mdfmt

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

.PHONY: mdfmt
mdfmt: ## Reformats Markdown files with markdownfmt.
mdfmt: $(MARKDOWNFMT)
	$(MARKDOWNFMT) -w $(MDFMT_FILES)

.PHONY: check-mdfmt
check-mdfmt: ## Verifies that all Markdown files are properly formatted.
check-mdfmt: $(MARKDOWNFMT)
	@DIFF=$$($(MARKDOWNFMT) -d $(MDFMT_FILES)); \
	if [ -n "$$DIFF" ]; then \
		echo "--- mdfmt would change:"; \
		echo "$$DIFF"; \
		echo "Run 'make mdfmt' to fix"; \
		exit 1; \
	fi

$(MARKDOWNFMT): $(SRC_FILES)
	go build -o $@ ./cmd/markdownfmt
