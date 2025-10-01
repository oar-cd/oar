.PHONY: lint test test-verbose test-one test-ci templ templ-watch tailwind tailwind-watch generate air build dev release release-patch release-minor release-major build-release

.EXPORT_ALL_VARIABLES:

CGO_ENABLED = 1
OAR_VERSION ?= $(shell git rev-parse --short HEAD)
OAR_EXECUTABLE_FILENAME ?= oar
OAR_WEB_ASSETS_FILENAME ?= web-assets.tar.gz
OAR_BUILD_ARTIFACTS_DIR ?= dist

lint:
	golangci-lint run --fix

test:
	gotestsum --format testname ./...

test-verbose:
	gotestsum --format standard-verbose -- -v -count=1 ./...

test-one:
	@if [ -z "$(TEST)" ]; then \
		echo "Usage: make test-one TEST=TestName"; \
		exit 1; \
	fi
	gotestsum --format standard-verbose -- -v -count=1 -run "^$(TEST)$$" ./...

test-ci:
	go run gotest.tools/gotestsum@latest --format testname -- -coverprofile=coverage.txt ./...

templ:
	templ generate

templ-watch:
	templ generate --watch

tailwind:
	tailwindcss -i ./web/assets/css/input.css -o ./web/assets/css/output.css

tailwind-watch:
	tailwindcss -i ./web/assets/css/input.css -o ./web/assets/css/output.css --watch

generate: tailwind templ

air:
	air -c .air.toml

build:
	go build -ldflags="-s -w -X github.com/oar-cd/oar/app.Version=$(OAR_VERSION)" -o ./${OAR_BUILD_ARTIFACTS_DIR}/$(OAR_EXECUTABLE_FILENAME) .

assets:
	tar -czf ./${OAR_BUILD_ARTIFACTS_DIR}/${OAR_WEB_ASSETS_FILENAME} web/assets

build-release: build assets

dev:
	make -j4 tailwind-watch templ-watch air

release:
	@echo "Available release types:"
	@echo "  make release-patch  # Patch version (x.y.Z)"
	@echo "  make release-minor  # Minor version (x.Y.0)"
	@echo "  make release-major  # Major version (X.0.0)"

release-patch:
	./release.sh patch

release-minor:
	./release.sh minor

release-major:
	./release.sh major
