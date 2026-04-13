.PHONY: build test swagger run up up-fg down logs

build:
	go build -o bin/subscription-service ./cmd/subscription-service

test:
	go test ./...

swagger:
	go generate ./cmd/subscription-service

run:
	go run ./cmd/subscription-service

# Запуск стека (PostgreSQL + приложение) в фоне с пересборкой образа приложения
up:
	docker compose up --build -d

# То же, но с логами в текущем терминале (foreground)
up-fg:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f
