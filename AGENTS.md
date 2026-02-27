# Repository Guidelines

## Project Structure & Module Organization
The service entrypoint is `main.go`, with HTTP/MCP orchestration in top-level server files (for example `mcp_server.go`, `handlers_api.go`, `service.go`). Core architecture follows layered Go packages under `internal/`:
- `internal/app/*`: use cases
- `internal/domain/*`: domain models/rules
- `internal/infra/*`: Playwright/browser, selectors, config, gateways
- `internal/interfaces/wiring`: dependency wiring

Utility binaries live in `cmd/`. End-to-end browser tests are in `integration/`, while unit tests are colocated as `*_test.go`. Runtime/config assets are in `configs/`, `docker/`, `docs/`, and `assets/`.

## Build, Test, and Development Commands
- `go mod download`: fetch Go dependencies.
- `go run github.com/playwright-community/playwright-go/cmd/playwright install chromium`: install browser runtime.
- `make run`: run locally with headed browser (`--headless=false`).
- `make run-prod`: run in headless mode.
- `make build` / `make build-all`: build local or cross-platform binaries.
- `make test` (or `go test -v ./...`): run full test suite.
- `go test ./integration/... -v -timeout 120s`: run integration tests (requires valid cookies/network).
- `make fmt`: apply `go fmt` + `gofmt`.

## Coding Style & Naming Conventions
Target Go `1.24` and keep code `gofmt`-clean before PRs. Use idiomatic Go naming: package names lowercase, exported symbols `PascalCase`, unexported symbols `camelCase`. Follow existing file naming patterns like `login_session_playwright.go`. Keep handlers thin; place business logic in `internal/app` and platform/browser details in `internal/infra`.

## Testing Guidelines
Use `testing` with `testify` (`require`/`assert`) where helpful. Name tests as `Test<Behavior>`. Prefer fast unit tests by mocking gateways (`internal/app/testkit`). Run integration tests only when cookies/session are available; write-path tests may mutate account state.

## Commit & Pull Request Guidelines
Recent history follows conventional prefixes: `feat:`, `fix:`, `refactor:`, `chore:`, `ci:` (imperative, concise subject). For PRs, include:
- what changed and why
- test commands run
- related issue(s)
- API/tooling docs updates when behavior changes (`docs/API.md`, Swagger docs)

## Security & Configuration Tips
Do not commit real account cookies or private data. Use `COOKIES_PATH` for local overrides and Docker volume mounts under `docker/` for persistence.
