.PHONY: lint-go
lint-go:
	golangci-lint run -v --config .golangci.yml

.PHONY: lint
lint: lint-go

.PHONY: goimports
goimports:
	gofancyimports fix --local github.com/c4t-but-s4d/neo -w $(shell find . -type f -name '*.go' -not -path "./proto/*")

.PHONY: test
test:
	go test -race -timeout 1m ./...

.PHONY: validate
validate: lint test
