include .env
MIGRATIONS_PATH ?= ./cmd/migrate/migrations

.PHONY: migration
migration:
	@migrate create -seq -ext sql -dir "$(MIGRATIONS_PATH)" "$(name)"

.PHONY: migrate-up
migrate-up:
	@migrate -path="$(MIGRATIONS_PATH)" -database="$(DB_ADDR)" up

.PHONY: migrate-down
migrate-down:
	@migrate -path="$(MIGRATIONS_PATH)" -database="$(DB_ADDR)" down 1

.PHONY: migrate-drop
migrate-drop:
	@migrate -path="$(MIGRATIONS_PATH)" -database="$(DB_ADDR)" drop -f

.PHONY: db/up db/down
db/up:
	docker compose up -d

db/down:
	docker compose down
