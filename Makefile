.PHONY: lint test test_ci templ templ_watch server tailwind tailwind_watch generate dockerimage dev build build-dev release release-patch release-minor release-major

lint:
	golangci-lint run --fix

test:
	gotestsum ./...

test_ci:
	go run gotest.tools/gotestsum@latest -- -coverprofile=coverage.txt ./...

templ:
	templ generate

templ_watch:
	templ generate --watch

air:
	air \
	--build.cmd "go build -o tmp/bin/main ." \
	--build.bin "tmp/bin/main" \
	--build.delay "100" \
	--build.exclude_dir "cmd,dev_oar_data_dir" \
	--build.include_ext "go" \
	--build.stop_on_error "false" \
	--misc.clean_on_exit true

tailwind:
	tailwindcss -i ./frontend/assets/css/input.css -o ./frontend/assets/css/output.css

tailwind_watch:
	tailwindcss -i ./frontend/assets/css/input.css -o ./frontend/assets/css/output.css --watch

server:
	go run .

generate: tailwind templ

dockerimage:
	docker build -t oar .

up:
	docker compose up

down:
	docker compose down

build:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION must be set. Usage: make build VERSION=1.2.3"; \
		exit 1; \
	fi
	CGO_ENABLED=1 go build -ldflags="-s -w -X github.com/ch00k/oar/cmd/version.CLIVersion=$(VERSION)" -o oar ./cmd

build-dev:
	CGO_ENABLED=1 go build -ldflags="-X github.com/ch00k/oar/cmd/version.CLIVersion=dev-$(shell git rev-parse --short HEAD)" -o oar ./cmd

dev:
	make -j3 tailwind_watch templ_watch air

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
