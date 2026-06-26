GO ?= go
GOLANGCI_LINT ?= golangci-lint
IMAGE ?= kuafu:lab

.PHONY: help fmt fmt-check test test-cover build docker-build lint validate

help:
	@echo "Kuafu development targets"
	@echo "  make fmt          Format Go code"
	@echo "  make fmt-check    Check Go formatting"
	@echo "  make test         Run unit tests"
	@echo "  make test-cover   Run tests with coverage"
	@echo "  make build        Build server and CLI"
	@echo "  make docker-build Build Docker image"
	@echo "  make lint         Run golangci-lint"
	@echo "  make validate     Run fmt-check, test, build"

fmt:
	$(GO)fmt -w cmd internal pkg

fmt-check:
	@test -z "$$($(GO)fmt -l cmd internal pkg)" || (echo "gofmt needed:" && $(GO)fmt -l cmd internal pkg && exit 1)

test:
	$(GO) test ./...

test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

build:
	mkdir -p bin
	$(GO) build -o bin/kuafu-server ./cmd/kuafu-server
	$(GO) build -o bin/kuafu ./cmd/kuafu

docker-build:
	docker build -t $(IMAGE) .

lint:
	$(GOLANGCI_LINT) run

validate: fmt-check test build