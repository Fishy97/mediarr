.PHONY: test test-backend test-frontend build docker-up docker-down

test: test-backend test-frontend

test-backend:
	cd backend && go test ./...

test-frontend:
	npm --prefix frontend run test -- --run

build:
	npm --prefix frontend run build
	cd backend && go build ./cmd/mediarr

docker-up:
	docker compose up --build

docker-down:
	docker compose down
