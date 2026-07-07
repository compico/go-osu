.PHONY: init deps build build-prod frontend-build frontend-dev run clean

## Bootstrap: install required dev tools
init:
	go mod download

## Install frontend dependencies (npm)
deps:
	npm install

## Build a debug binary (no compiled frontend assets required)
build:
	go build -o bin/goosu ./cmd/

## Build the production binary. Requires the frontend to be built first (make frontend-build),
## since the Go server reads compiled assets from frontend/dist/.vite/manifest.json at runtime.
build-prod: frontend-build
	go build -o bin/goosu ./cmd/

## Build the frontend production bundle into frontend/dist
frontend-build:
	npm run build

## Run the Vite dev server (HMR). Keep this running in a separate terminal
## while using `make run` below.
frontend-dev:
	npm run dev

## Run the Go backend in development mode.
## Requires `make frontend-dev` running in another terminal (config.yml: app.env=development).
## Database migrations are applied automatically on startup, no manual step needed.
run:
	go run ./cmd/ http --config config.yml

## Remove build artifacts
clean:
	rm -rf bin frontend/dist
