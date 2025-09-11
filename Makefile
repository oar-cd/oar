.PHONY: lint test test_ci templ templ_watch web_server tailwind tailwind_watch generate docker_image dev build_cli release release_patch release_minor release_major

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
	air \
	--build.cmd "go build -o tmp/bin/main ./web" \
	--build.bin "tmp/bin/main" \
	--build.delay "100" \
	--build.exclude_dir "cmd,dev_oar_data_dir" \
	--build.include_ext "go" \
	--build.stop_on_error "false" \
	--misc.clean_on_exit true

web_server:
	go run ./web

docker_image:
	docker build --build-arg VERSION=dev-$(shell git rev-parse --short HEAD) -t oar .

build_cli:
	CGO_ENABLED=1 go build -ldflags="-X github.com/oar-cd/oar/cmd/version.CLIVersion=dev-$(shell git rev-parse --short HEAD)" -o oar-cli ./cmd

dev:
	make -j3 tailwind_watch templ_watch air

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
