.PHONY: help up down logs be-test fe-test proto be-run fe-run

help:
	@echo "Targets:"
	@echo "  up        - docker compose up --build"
	@echo "  down      - docker compose down -v"
	@echo "  logs      - tail compose logs"
	@echo "  be-test   - go test ./... (backend)"
	@echo "  fe-test   - npm test (frontend)"
	@echo "  proto     - regenerate telemetry gRPC stubs"
	@echo "  be-run    - run backend against local postgres/redis"
	@echo "  fe-run    - run frontend dev server"

up:
	docker compose up --build

down:
	docker compose down -v

logs:
	docker compose logs -f --tail=100

be-test:
	cd backend && go test ./...

fe-test:
	cd frontend && npm test

proto:
	cd backend && protoc --go_out=. --go_opt=module=github.com/The-Christopher-Robin/pulse-front/backend \
	                    --go-grpc_out=. --go-grpc_opt=module=github.com/The-Christopher-Robin/pulse-front/backend \
	                    proto/telemetry.proto

be-run:
	cd backend && go run ./cmd/server

fe-run:
	cd frontend && npm run dev
