# Rules for AI coding agents

Everyone on this team uses AI assistants. This file keeps what they write consistent. Read it before changing code. If a rule here is wrong, change this file in a PR, do not work around it.

These are the rules for the whole repository. Each part adds its own, read them before changing anything there:

- `backend/`: [`backend/AGENTS.md`](backend/AGENTS.md)
- `frontend/`: [`frontend/AGENTS.md`](frontend/AGENTS.md)

Read first: `docs/ARCHITECTURE.md` (layers, conventions), `docs/DOMAIN.md` (business rules), `docs/DECISIONS.md` (why), `TEAM.md` (who owns what). The API contract is `api/openapi.yaml`.

## Stack and commands

Go backend, React + TypeScript + Vite frontend, PostgreSQL, Docker Compose.

| Task | Command |
|------|---------|
| Run everything | `docker compose up --build --wait` |
| Smoke test | `scripts/smoke.sh` |
| Frontend | `cd frontend && npm ci && npm run build` |

Backend commands are in [`backend/AGENTS.md`](backend/AGENTS.md).

## API contract

Every new or changed endpoint is described in `api/openapi.yaml` in the same PR as the code, with its request, response and error cases. Swagger UI at `/docs/` must show it. Field names in the spec equal the JSON tags in Go. The HTTP conventions and error codes are in [`backend/AGENTS.md`](backend/AGENTS.md#http-conventions).

## Code style

- No comments that restate the code. Comment only the non-obvious why.
- Do not add libraries without a reason written in the PR.
- No secrets in code, logs or images.

## Git and PRs

- One issue per branch and PR. Branch `feat/<issue>-short-name`, `fix/...`, `docs/...`, `chore/...`, `ci/...`.
- Conventional commits: `feat: ...`, `fix: ...`, `docs: ...`, `chore: ...`, `ci: ...`, `test: ...`, `refactor: ...`. Lower case, no trailing period.
- PR body: what changed, how it was tested, `Closes #<issue>`. A PR stacked on another branch does not close its issue on merge, close it by hand.
- Run the checks before opening a PR. If you could not run something, say so in the PR.
- Do not add AI attribution lines (`Co-Authored-By`, "Generated with") to commits or PR text.
- Work only in the feature you own (`TEAM.md`). A change in someone else's package goes through an issue or a review request first.
- Do not push to `main`. Do not open a PR with half-finished work.
