.PHONY: frontend-deps test test-backend test-frontend vet build verify docker-validate docker-build docker-up docker-down ci

frontend-deps:
	npm --prefix frontend ci

test: test-backend test-frontend

test-backend:
	cd backend && go test ./...

test-frontend:
	npm --prefix frontend run test -- --run

vet:
	cd backend && go vet ./...

build:
	npm --prefix frontend run build
	cd backend && go build ./cmd/mediarr

verify:
	scripts/verify-no-delete.sh

docker-validate:
	docker compose config --quiet
	docker compose --profile ai config --quiet

docker-build:
	docker compose build mediarr

docker-up:
	docker compose up --build

docker-down:
	docker compose down

ci: frontend-deps test vet build verify docker-validate docker-build
