.PHONY: init deps build build-prod build-linux build-windows frontend-build frontend-dev run clean install-hooks fetch-refs test-reference

install-hooks:
	chmod +x .githooks/pre-commit
	git config --local core.hooksPath .githooks

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

# Downloads the .osu files of the skills reference test into pkg/osu/skills/test_data
# (git ignores them). Files that are already there are kept.
fetch-refs:
	go run ./internal/tools/fetchrefs

# Same check as the reference job in CI: fails instead of skipping when a map is missing.
test-reference: fetch-refs
	REQUIRE_REFERENCE=1 go test ./pkg/osu/skills -run TestSkillsValues_CompareWithReference -count=1 -v
