.PHONY: lint test test_ci templ templ_watch server tailwind tailwind_watch generate dockerimage dev

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

server:
	air \
	--build.cmd "go build -o tmp/bin/main ./main.go" \
	--build.bin "tmp/bin/main" \
	--build.delay "100" \
	--build.exclude_dir "node_modules" \
	--build.include_ext "go" \
	--build.stop_on_error "false" \
	--misc.clean_on_exit true

tailwind:
	tailwindcss -i ./ui/assets/css/input.css -o ./ui/assets/css/output.css

tailwind_watch:
	tailwindcss -i ./ui/assets/css/input.css -o ./ui/assets/css/output.css --watch

generate: tailwind templ

dockerimage:
	docker build -t oar .

up:
	docker compose up

down:
	docker compose down

dev:
	make -j3 tailwind_watch templ_watch server
