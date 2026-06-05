.PHONY: all up down restart logs clean test mod-tidy

all: test

# Docker Compose Commands
up:
	docker-compose up -d

down:
	docker-compose down

restart: down up

logs:
	docker-compose logs -f

clean:
	docker-compose down -v

# Go Commands
test:
	go test -v -race ./...

mod-tidy:
	go mod tidy

# Database & Redis Variables
DATABASE_URL ?= "postgresql://aegis_user:aegis_password@localhost:5432/aegis"
REDIS_ADDR ?= "localhost:6379"
REDIS_PASS ?= "aegis_redis_pass"
NATS_URL ?= "nats://localhost:4222"

# Proxy Commands
run-proxy:
	go run ./cmd/proxy -db=$(DATABASE_URL) -redis-addr=$(REDIS_ADDR) -redis-pass=$(REDIS_PASS) -nats-url=$(NATS_URL)

build-proxy:
	go build -o bin/proxy ./cmd/proxy

# Control Plane Commands
run-control-plane:
	go run ./cmd/control-plane -db=$(DATABASE_URL)

build-control-plane:
	go build -o bin/control-plane ./cmd/control-plane

# Analytics Worker Commands
run-analytics-worker:
	go run ./cmd/analytics-worker -db=$(DATABASE_URL) -nats-url=$(NATS_URL)

build-analytics-worker:
	go build -o bin/analytics-worker ./cmd/analytics-worker

# Proto Generation
# Requires protoc to be installed locally (e.g. `choco install protoc`)
generate-proto:
	@echo "Installing Go proto plugins..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
	@echo "Generating Go protos..."
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/proto/analyzer.proto
	@echo "Generating Python protos..."
	cd ai-engine && python -m grpc_tools.protoc -I../internal/proto --python_out=. --pyi_out=. --grpc_python_out=. ../internal/proto/analyzer.proto

# Python AI Engine Commands
setup-ai:
	cd ai-engine && python -m venv venv
	@echo "Virtual environment created. Please activate it and run: pip install -r ai-engine/requirements.txt"

run-ai:
	cd ai-engine && python main.py

# Database Migrations
# To use Neon DB, run: make migrate-up DATABASE_URL="postgres://user:pass@ep-rest-of-host.neon.tech/dbname?sslmode=require"
DATABASE_URL ?= "postgresql://aegis_user:aegis_password@localhost:5432/aegis"

migrate-up:
	go run ./cmd/migrate -up -db=$(DATABASE_URL)

migrate-down:
	go run ./cmd/migrate -down -db=$(DATABASE_URL)
