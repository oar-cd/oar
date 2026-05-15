.PHONY: lint lint-fix test test-verbose test-one test-ci templ templ-watch tailwind tailwind-watch generate air build dev release release-patch release-minor release-major build-release

.EXPORT_ALL_VARIABLES:

CGO_ENABLED = 1
OAR_VERSION ?= $(shell git rev-parse --short HEAD)
OAR_EXECUTABLE_FILENAME ?= oar
OAR_WEB_ASSETS_FILENAME ?= web-assets.tar.gz
OAR_BUILD_ARTIFACTS_DIR ?= dist

# Bump this to upgrade the linter. To re-install, delete ./bin/golangci-lint.
# The upstream install script has a checksum-matching bug when the release
# ships an SBOM sidecar, so we download the tarball directly and verify the
# SHA-256 against the published checksums file.
GOLANGCI_LINT_VERSION ?= 2.12.2
GOLANGCI_LINT := ./bin/golangci-lint

$(GOLANGCI_LINT):
	@mkdir -p ./bin
	@tmp=$$(mktemp -d) && \
		os=$$(uname -s | tr A-Z a-z) && \
		arch=$$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/') && \
		name="golangci-lint-$(GOLANGCI_LINT_VERSION)-$${os}-$${arch}" && \
		base="https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)" && \
		echo "Downloading $${name}.tar.gz" && \
		curl -fsSL -o "$${tmp}/archive.tar.gz" "$${base}/$${name}.tar.gz" && \
		curl -fsSL -o "$${tmp}/sums.txt" "$${base}/golangci-lint-$(GOLANGCI_LINT_VERSION)-checksums.txt" && \
		(cd $${tmp} && awk -v n="$${name}.tar.gz" '$$2==n {print $$1"  archive.tar.gz"}' sums.txt | sha256sum -c -) && \
		tar -xzf "$${tmp}/archive.tar.gz" -C "$${tmp}" "$${name}/golangci-lint" && \
		mv "$${tmp}/$${name}/golangci-lint" $(GOLANGCI_LINT) && \
		rm -rf "$${tmp}"

lint: $(GOLANGCI_LINT)
	CGO_ENABLED=0 $(GOLANGCI_LINT) run

lint-fix: $(GOLANGCI_LINT)
	CGO_ENABLED=0 $(GOLANGCI_LINT) run --fix

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

run:
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
