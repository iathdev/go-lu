-include .env

.PHONY: run build docker-up docker-down migrate-up migrate-down migrate-reset migrate-install

MIGRATE ?= migrate
STEPS ?= 1
URL := postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)
PATH_MIG := migrations

run:
	go run cmd/api/main.go

build:
	go build -o bin/api cmd/api/main.go

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate-up:
	@$(MIGRATE) -path "$(PATH_MIG)" -database "$(URL)" up

migrate-down:
	@if [ $(STEPS) -gt 3 ]; then echo "Error: max 3 steps allowed (got $(STEPS))"; exit 1; fi
	@$(MIGRATE) -path "$(PATH_MIG)" -database "$(URL)" down $(STEPS)

migrate-down-%:
	@$(MAKE) migrate-down STEPS=$*

migrate-force:
	@$(MIGRATE) -path "$(PATH_MIG)" -database "$(URL)" force $(VERSION)

migrate-reset:
	@$(MIGRATE) -path "$(PATH_MIG)" -database "$(URL)" drop -f

