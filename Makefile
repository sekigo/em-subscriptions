.PHONY: help run build test test-cover vet tidy swagger docker-up docker-down docker-logs migrate-create

BINARY := bin/em-subscriptions

help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Run the API locally (expects postgres running)
	go run ./cmd/api

build: ## Build a static binary into bin/
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/api

test: ## Run unit tests
	go test ./... -race -count=1

test-cover: ## Run tests with coverage
	go test ./... -race -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

vet: ## go vet
	go vet ./...

tidy: ## go mod tidy
	go mod tidy

swagger: ## Regenerate Swagger docs (requires `go install github.com/swaggo/swag/cmd/swag@latest`)
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

docker-up: ## Bring up postgres + api via docker compose
	docker compose up -d --build

docker-down: ## Stop and remove compose stack (keeps the volume)
	docker compose down

docker-logs: ## Tail api logs
	docker compose logs -f api

migrate-create: ## Create a new pair of migration files: make migrate-create name=add_thing
	@test -n "$(name)" || (echo "usage: make migrate-create name=<slug>" && exit 1)
	@NEXT=$$(printf "%06d" $$(( $$(ls migrations 2>/dev/null | sed -n 's/^\([0-9]*\)_.*/\1/p' | sort -n | tail -1 | sed 's/^0*//' 2>/dev/null) + 1 ))); \
	touch migrations/$${NEXT}_$(name).up.sql migrations/$${NEXT}_$(name).down.sql; \
	echo "created migrations/$${NEXT}_$(name).{up,down}.sql"
