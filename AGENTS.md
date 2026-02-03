# AGENTS.md

## Commands
- Format: `golangci-lint fmt`
- Lint: `golangci-lint run`
- Test all: `go test -race ./...`
- Test single: `go test ./internal/cache -run TestMountRequest_EnabledModes`

## Code Style
- Go 1.25+, use `any` instead of `interface{}`
- Imports: standard → external → `github.com/namespacelabs/spacectl` (enforced by gci)
- Formatting: gofumpt with extra rules enabled
- Errors: wrap with `fmt.Errorf("context: %w", err)`
- Logging: use `log/slog` (not log package)
- Tests: use `_test` package suffix to make sure we test the public API
