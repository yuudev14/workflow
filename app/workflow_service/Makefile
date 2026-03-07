#  make test ENV=".env.test"
export

.PHONY: migrate
.PHONY: migrate-down
.PHONY: test
.PHONY: swag


DATABASE_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST)/$(DB_NAME)?sslmode=disable
MIGRATIONS_PATH := file://db/migrations

migrate:
		migrate -source $(MIGRATIONS_PATH) -database $(DATABASE_URL) up

migrate-down:
		migrate -source $(MIGRATIONS_PATH) -database $(DATABASE_URL) down

# UNIT_TESTS_PKGS := $(shell go list ./... | grep -vE 'mocks|repository.go')
# UNIT_TESTS_COVERPKG := $(shell go list ./... | grep -vE 'mocks|repository.go' | tr '\n' ',')

# unit-test:
# 	go test $(UNIT_TESTS_PKGS) -coverpkg=$(UNIT_TESTS_COVERPKG) -coverprofile=coverage.out
# 	go tool cover -func=coverage.out


TESTS_PKGS := $(shell go list ./... | grep -vE 'mocks|grpc|repository.go')
TESTS_COVERPKG := $(shell go list ./... | grep -vE 'mocks|grpc|repository.go' | tr '\n' ',')

test:
	go test $(TESTS_PKGS) -coverpkg=$(TESTS_COVERPKG) -coverprofile=coverage.out
	go tool cover -func=coverage.out

view-coverage-html:
	go tool cover -html=coverage.out

swag:
	swag init
