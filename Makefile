.PHONY: init wire build run dev

## Bootstrap: install required dev tools
init:
	go install github.com/goforj/wire/cmd/wire@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest

## Run wire code generation
wire:
	wire ./internal/app/

## Build the binary
build:
	go build -o bin/sclient ./cmd/

## Build production binary (embeds frontend dist)
build-prod:
	go build -tags prod -o bin/sclient ./cmd/

## Run in development mode (expects `npm run dev` already running in ./frontend)
run:
	go run ./cmd/ http --config config.yml
