.DEFAULT_GOAL := help

BINARY      ?= olympics-api
BIN_DIR     ?= bin
MAIN        ?= ./cmd/main.go
OUT         ?= $(BIN_DIR)/$(BINARY)
MIGRATE     := go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1
MIGRATIONS  := internal/config/migrations
BUILD_FLAGS ?=
LDFLAGS     ?=

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the API locally
	go run $(MAIN)

.PHONY: build
build: ## Build the binary (override OUT/MAIN/BUILD_FLAGS/LDFLAGS for cross-compiles)
	@mkdir -p $(dir $(OUT))
	go build $(BUILD_FLAGS) $(if $(LDFLAGS),-ldflags="$(LDFLAGS)",) -o $(OUT) $(MAIN)

.PHONY: verify
verify: ## Verify go module dependencies
	go mod verify

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: tidy
tidy: ## Tidy go modules
	go mod tidy

.PHONY: migrate-create
migrate-create: ## Create a new migration pair: make migrate-create NAME=add_foo
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS) -seq $(NAME)

.PHONY: migrate-up
migrate-up: ## Apply migrations (DATABASE_URL=postgres://user:pass@host:port/db?sslmode=disable)
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration (set DATABASE_URL)
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" down 1

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)
