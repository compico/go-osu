.PHONY: init deps build build-prod build-linux build-windows frontend-build frontend-dev run clean

init:
	go mod download

deps:
	npm install

build:
	go build -o bin/goosu ./cmd/

frontend-build:
	npm run build

build-prod: frontend-build
	go build -o bin/goosu ./cmd/

build-linux: frontend-build
	GOOS=linux GOARCH=amd64 go build -o bin/goosu-linux-amd64 ./cmd/

build-windows: frontend-build
	GOOS=windows GOARCH=amd64 go build -o bin/goosu-windows-amd64.exe ./cmd/

frontend-dev:
	npm run dev

run:
	go run ./cmd/ http --config config.yml

clean:
	rm -rf bin frontend/dist