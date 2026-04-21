.PHONY: up down logs migrate-up migrate-down migrate-new backend-test frontend-dev frontend-test frontend-typecheck smoke

up: ; docker compose up --build
down: ; docker compose down
logs: ; docker compose logs -f

migrate-up: ; cd backend && go run ./cmd/migrate up
migrate-down: ; cd backend && go run ./cmd/migrate down 1
migrate-new: ; cd backend && migrate create -ext sql -dir migrations -seq $(name)

backend-test: ; cd backend && go test ./...
frontend-dev: ; cd frontend && pnpm dev
frontend-test: ; cd frontend && pnpm test
frontend-typecheck: ; cd frontend && pnpm typecheck

smoke: ; bash scripts/smoke.sh
