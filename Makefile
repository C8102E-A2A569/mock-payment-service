# Mock Payment Service — команды сборки и разработки

.PHONY: proto deps run test lint migrate-up migrate-down docker-build docker-up docker-down

# Генерация Go-кода из .proto (нужны: protoc, protoc-gen-go, protoc-gen-go-grpc)
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/payment/payment.proto

# Установка инструментов для генерации proto (один раз)
proto-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Загрузка зависимостей
deps:
	go mod download
	go mod tidy

# Запуск сервиса (локально; нужны Postgres, Kafka, Redis)
run:
	go run ./cmd/server

# Тесты
test:
	go test ./...
# Интеграционный тест репозитория (нужен Docker)
test-integration:
	go test -tags=integration ./internal/repository/postgres/...

# Линтер
lint:
	golangci-lint run

# Миграции (требуется настроенный конфиг и доступная БД)
migrate-up:
	go run ./cmd/migrate
migrate-down:
	go run ./cmd/migrate -down

# Docker
docker-build:
	docker build -f deployments/docker/Dockerfile -t mock-payment:latest .
docker-up:
	docker-compose -f deployments/docker-compose.yml up -d
docker-down:
	docker-compose -f deployments/docker-compose.yml down
