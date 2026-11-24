# Rules for AI coding agents

Everyone on this team uses AI assistants. This file keeps what they write consistent. Read it before changing code. If a rule here is wrong, change this file in a PR, do not work around it.

Read first: `docs/ARCHITECTURE.md` (layers, conventions), `docs/DOMAIN.md` (business rules), `docs/DECISIONS.md` (why), `backend/docs/DATA_MODEL.md` (tables), `TEAM.md` (who owns what). The API contract is `api/openapi.yaml`.

## Stack and commands

Go backend, React + TypeScript + Vite frontend, PostgreSQL, Docker Compose.

| Task | Command |
|------|---------|
| Run everything | `docker compose up --build --wait` |
| Backend checks | `cd backend && gofmt -l . && go vet ./... && go test -race ./...` |
| Database tests | need `DATABASE_URL`, CI runs them against PostgreSQL |
| Smoke test | `scripts/smoke.sh` |
| Frontend | `cd frontend && npm ci && npm run build` |

## One feature, one shape (backend)

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
7. Wire new features only in `cmd/server/main.go` and `internal/httpapi/router.go`. Protected routes go through `auth.RequireAuth`.
8. Fakes live only in `_test.go` files. Production code never contains an in-memory repository.

## HTTP conventions

- JSON, `snake_case`. Create returns 201, other success 200.
- Errors are always `httpx.WriteError(w, status, code, message)`, body `{"code","message"}`.
- Handler error mapping, one `write<Feature>Error` function per handler:

| Error | Status | `code` |
|-------|--------|--------|
| invalid JSON | 400 | `invalid_body` |
| `ErrValidation` | 400 | `validation_error` |
| missing or bad token | 401 | `unauthorized` |
| `ErrNotFound` (also other shop's row) | 404 | `not_found` |
| duplicate or state conflict | 409 | feature code, for example `sku_taken`, `email_taken`, `insufficient_stock` |
| anything else | 500 | `internal`, message never leaks details |

- Lists: `?page=1&page_size=20`, default 20, max 100. Response `{"items": [], "page", "page_size", "total"}`, `items` is `[]`, never `null`.
- Use `httpx.DecodeJSON` for bodies (unknown fields rejected, 1 MB limit).

## Database

- New table or column means a new migration `NNNN_name.up.sql` and `.down.sql` in `backend/migrations`. Never edit a merged migration.
- Update `backend/docs/schema.dbml` and `backend/docs/DATA_MODEL.md` in the same PR.
- Constraints for rules that must never break (positive price, stock bounds, unique SKU per shop).

## API contract

Every new or changed endpoint is described in `api/openapi.yaml` in the same PR as the code, with its request, response and error cases. Swagger UI at `/docs/` must show it. Field names in the spec equal the JSON tags in Go.

## Code style

- `gofmt`, `go vet` clean. Short names for short scopes, full words for exported names.
- No comments that restate the code. Comment only the non-obvious why, and keep exported doc comments to one line.
- Return errors with `fmt.Errorf("what failed: %w", err)`. Compare with `errors.Is`.
- Do not add libraries without a reason written in the PR. Standard library first.
- No secrets in code, logs or images. Config comes from environment variables in `internal/config`.
- Frontend: feature folders under `src/features/<feature>`, server state with TanStack Query, API types generated from `api/openapi.yaml`, every screen has loading, empty and error states.

## Tests

- Service: table-driven tests with `memRepository`, including the "other shop gets not found" case.
- Repository: real PostgreSQL, covers constraints and unique violations.
- Handler: `httptest`, covers 400, 401, 404 and the success shape.
- Run the checks above before opening a PR. If you could not run something, say so in the PR.

## Git and PRs

- One issue per branch and PR. Branch `feat/<issue>-short-name`, `fix/...`, `docs/...`, `chore/...`, `ci/...`.
- Conventional commits: `feat: ...`, `fix: ...`, `docs: ...`, `chore: ...`, `ci: ...`, `test: ...`, `refactor: ...`. Lower case, no trailing period.
- PR body: what changed, how it was tested, `Closes #<issue>`. A PR stacked on another branch does not close its issue on merge, close it by hand.
- Do not add AI attribution lines (`Co-Authored-By`, "Generated with") to commits or PR text.
- Work only in the feature you own (`TEAM.md`). A change in someone else's package goes through an issue or a review request first.
- Do not push to `main`. Do not open a PR with half-finished work.
