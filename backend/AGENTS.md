# Backend rules for AI coding agents

These rules add to the root [`AGENTS.md`](../AGENTS.md) for everything under `backend/`. If a rule here is wrong, change this file in a PR, do not work around it.

Read first: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) (layers, conventions), [`docs/DATA_MODEL.md`](docs/DATA_MODEL.md) (tables).

## Commands

| Task | Command |
|------|---------|
| Backend checks | `cd backend && gofmt -l . && go vet -tags e2e ./... && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run --build-tags=e2e && go test -race ./...` |
| End-to-end tests | `cd backend && go test -race -tags e2e ./e2e/...` (needs `TEST_DATABASE_URL`) |
| Database tests | need `TEST_DATABASE_URL` and CI runs them against PostgreSQL. It must name a throwaway database ending in `_test` (`seller_test`), never the one the app uses: the tests truncate tables and refuse other names. New database tests use `dbtest.NewPool(t, "<feature>")`, not the URL directly |

## One feature, one shape

A feature is a package `backend/internal/<feature>` with these files and no others unless needed:

| File | Content |
|------|---------|
| `model.go` | types and request/response structs, JSON tags in `snake_case` |
| `errors.go` | sentinel errors: `ErrNotFound`, `ErrValidation`, plus feature errors |
| `repository.go` | `Repository` interface only, declared by the feature |
| `pgrepository.go` | `PgRepository` + `NewPgRepository(pool)`, hand-written pgx SQL |
| `service.go` | business rules and validation, no `net/http`, no SQL |
| `handler.go` | `Handler` + `NewHandler(svc)`: parse, call service, map errors |
| `*_test.go` | service tests with `memRepository` in `memrepository_test.go`, database tests in `pgrepository_test.go`, handler tests with `httptest` |

Rules:
1. Layers go one way: handler, service, repository. Features never import each other's concrete types.
2. Every repository method takes `shopID` first after `ctx` and filters by it. Another shop's row is `ErrNotFound`, never 403. `auth` (runs before a shop exists) and `catalog` (shared reference data) are the exceptions.
3. The shop comes from `auth.MustPrincipal(w, r)` in the handler, never from the body, query or URL.
4. Validate and trim in the service before any SQL. Database `CHECK` and `UNIQUE` constraints are the second line.
5. Money is `int64` in so'm. No floats. Time is UTC `timestamptz`.
6. Anything that changes more than one row uses `database.InTx`. New repositories accept `database.DBTX`.
7. Wire new features only in `internal/app/app.go` (services and handlers) and `internal/httpapi/router.go` (routes). `cmd/server/main.go` only starts the process. Protected routes are registered with `protected(handler)`, which applies `auth.RequireAuth` and the per shop rate limit. Do not register a protected route any other way.
8. Fakes live only in `_test.go` files. Production code never contains an in-memory repository.

## HTTP conventions

- JSON, `snake_case`. Create returns 201, other success 200.
- Errors are always `httpx.WriteError(w, status, code, message)`, body `{"code","message"}`. Unknown routes and wrong methods get the same shape from `httpx.JSONErrors`, so never write a plain-text error.
- Unset optional values follow the column: nullable column is `null`, `NOT NULL DEFAULT ''` text is `""`, lists are `[]`. Times are UTC, call `.UTC()` after scanning a `timestamptz`.
- Handler error mapping, one `write<Feature>Error` function per handler:

| Error | Status | `code` |
|-------|--------|--------|
| invalid JSON | 400 | `invalid_body` |
| `ErrValidation` | 400 | `validation_error` |
| missing or bad token | 401 | `unauthorized` |
| `ErrNotFound` (also other shop's row) | 404 | `not_found` |
| duplicate or state conflict | 409 | feature code, for example `sku_taken`, `email_taken`, `insufficient_stock` |
| file over the size limit | 413 | `file_too_large` |
| wrong media type | 415 | `unsupported_media_type` |
| rate limit hit (`httpx.WriteRateLimited`) | 429 | `rate_limited`, with `Retry-After` |
| server busy, retry soon | 503 | `busy`, with `Retry-After` |
| anything else | 500 | `internal`, message never leaks details |

- Lists: `?page=1&page_size=20`, default 20, max 100. Response `{"items": [], "page", "page_size", "total"}`, `items` is `[]`, never `null`.
- Use `httpx.DecodeJSON` for bodies (unknown fields rejected, 1 MB limit).

## Database

- New table or column means a new migration `NNNN_name.up.sql` and `.down.sql` in `backend/migrations`. Never edit a merged migration.
- Update `backend/docs/schema.dbml` and `backend/docs/DATA_MODEL.md` in the same PR.
- Constraints for rules that must never break (positive price, stock bounds, unique SKU per shop).

## Code style

- `gofmt`, `go vet` clean. Short names for short scopes, full words for exported names.
- Keep exported doc comments to one line.
- Return errors with `fmt.Errorf("what failed: %w", err)`. Compare with `errors.Is`.
- Standard library first.
- Config comes from environment variables in `internal/config`.

## Tests

- Service: table-driven tests with `memRepository`, including the "other shop gets not found" case.
- Repository: real PostgreSQL, covers constraints and unique violations.
- Handler: `httptest`, covers 400, 401, 404 and the success shape.
- End-to-end: `backend/e2e`, the real router and PostgreSQL through `app.NewHandler`. A feature that adds routes also adds to the journey, the isolation test (another seller gets 404), the error cases and, if it is expensive or sensitive, a limit. Use the helpers in `e2e/helpers_test.go`, files carry `//go:build e2e`.
