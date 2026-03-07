.PHONY: build run test lint \
       frontend-install frontend-build frontend-dev frontend-lint \
       migrate seed \
       docker-build docker-run docker-stop \
       dev all clean

# Go backend
GO_DIR := backend-go
GO_BIN := $(GO_DIR)/bin/server
GO_CMD := $(GO_DIR)/cmd/server

build:
	cd $(GO_DIR) && CGO_ENABLED=0 go build -o bin/server ./cmd/server

run: build
	cd $(GO_DIR) && ./bin/server

test:
	cd $(GO_DIR) && go test ./...

lint:
	cd $(GO_DIR) && golangci-lint run ./...

# Frontend
frontend-install:
	cd frontend && npm install

frontend-build:
	cd frontend && npm run build

frontend-dev:
	cd frontend && npm run dev

frontend-lint:
	cd frontend && npm run lint

# Database
migrate:
	cd $(GO_DIR) && go run ./cmd/server -migrate

seed:
	@echo "Loading seed data from configs/examples/ ..."
	@if [ -d configs/examples ]; then \
		cd $(GO_DIR) && go run ./cmd/server -seed; \
	else \
		echo "No configs/examples/ directory found"; \
	fi

# Docker
docker-build:
	docker build -t ledgerline .

docker-run:
	docker compose up -d

docker-stop:
	docker compose down

# Combined
dev:
	@echo "Starting backend and frontend in dev mode..."
	@$(MAKE) -j2 _dev-backend _dev-frontend

_dev-backend:
	cd $(GO_DIR) && go run ./cmd/server

_dev-frontend:
	cd frontend && npm run dev

all: build frontend-build

clean:
	rm -rf $(GO_DIR)/bin
	rm -rf frontend/dist
