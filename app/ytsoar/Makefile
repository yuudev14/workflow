#  make test ENV=".env.test"
export

.PHONY: migrate
.PHONY: migrate-down
.PHONY: test
.PHONY: swag
.PHONY: harness-deps


-include .env

DATABASE_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable
MIGRATIONS_DIR := db/migrations

migrate:
		goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
		goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

sqlc:
		cd db && sqlc generate

# Then: export NODE_PATH=$(CURDIR)/$(HARNESS_DIR)/node_modules
HARNESS_DIR := internal/adapters/runtimes/localexec/harness
harness-deps:
		npm install --prefix $(HARNESS_DIR)

UNIT_TEST_PKGS := $(shell go list ./... | grep -vE '/mocks|/grpc')

unit-test:
	go test $(UNIT_TEST_PKGS) -coverpkg=./... -coverprofile=coverage.out
	grep -vE "repository.go" coverage.out > coverage_filtered.out
	go tool cover -func=coverage_filtered.out

# TESTS_PKGS := $(shell go list ./... | grep -vE 'mocks|grpc|repository.go')
# TESTS_COVERPKG := $(shell go list ./... | grep -vE 'mocks|grpc|repository.go' | tr '\n' ',')

test:
	go test $(TESTS_PKGS) -coverpkg=$(TESTS_COVERPKG) -coverprofile=coverage.out
	go tool cover -func=coverage.out

view-coverage-html:
	go tool cover -html=coverage.out

swag:
	swag init
