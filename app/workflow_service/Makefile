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

test:
	go test -coverprofile=coverage.out -v ./... 

view-coverage:
	go tool cover -html=coverage.out

swag:
	swag init
