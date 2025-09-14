.PHONY: lint test test_verbose test_ci templ templ_watch tailwind tailwind_watch generate air build dev release release_patch release_minor release_major

.EXPORT_ALL_VARIABLES:

CGO_ENABLED = 1
OAR_VERSION ?= $(shell git rev-parse --short HEAD)
OAR_EXECUTABLE_FILENAME ?= oar
OAR_BUILD_ARTIFACTS_DIR ?= dist

lint:
	golangci-lint run --fix

test:
	gotestsum --format testname ./...

test_verbose:
	gotestsum --format standard-verbose ./...

test_ci:
	go run gotest.tools/gotestsum@latest --format testname -- -coverprofile=coverage.txt ./...

templ:
	templ generate

templ_watch:
	templ generate --watch

tailwind:
	tailwindcss -i ./web/assets/css/input.css -o ./web/assets/css/output.css

tailwind_watch:
	tailwindcss -i ./web/assets/css/input.css -o ./web/assets/css/output.css --watch

generate: tailwind templ

air:
	air -c .air.toml

build:
	go build -ldflags="-s -w -X github.com/oar-cd/oar/cmd/version.Version=$(OAR_VERSION)" -o ./${OAR_BUILD_ARTIFACTS_DIR}/$(OAR_EXECUTABLE_FILENAME) .

dev:
	make -j4 tailwind_watch templ_watch air

release:
	@echo "Available release types:"
	@echo "  make release-patch  # Patch version (x.y.Z)"
	@echo "  make release-minor  # Minor version (x.Y.0)"
	@echo "  make release-major  # Major version (X.0.0)"

release_patch:
	./release.sh patch

release_minor:
	./release.sh minor

release_major:
	./release.sh major
