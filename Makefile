.PHONY: test test-backend test-backend-docker test-frontend lint lint-backend lint-frontend typecheck build build-backend build-frontend coverage up down

BACKEND  := backend
FRONTEND := frontend

## test: everything required by the definition of done
test: lint-backend test-backend typecheck lint-frontend test-frontend

test-backend:
	cd $(BACKEND) && go test -race -cover ./...

# Race-detector tests in the backend's Docker test stage, for hosts without cgo.
test-backend-docker:
	docker build --target test $(BACKEND)

lint-backend:
	@cd $(BACKEND) && unformatted="$$(gofmt -l .)" && \
		if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	cd $(BACKEND) && go vet ./...

test-frontend:
	cd $(FRONTEND) && npm test -- --run

typecheck:
	cd $(FRONTEND) && npx tsc -b --noEmit

lint-frontend:
	cd $(FRONTEND) && npm run lint

lint: lint-backend lint-frontend

build: build-backend build-frontend

build-backend:
	cd $(BACKEND) && go build ./...

build-frontend:
	cd $(FRONTEND) && npm run build

coverage:
	cd $(BACKEND) && go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
	cd $(FRONTEND) && npm run test:coverage

up:
	docker compose up --build

down:
	docker compose down
