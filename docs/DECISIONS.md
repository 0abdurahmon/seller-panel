# Decisions

Record every important decision here: what you chose, what else you considered, and why.

## D1 — Stack (Day 1)
- Frontend: React (TypeScript, Vite)
- Backend: Go (REST API)
- Database: PostgreSQL
- Hosting / run: Docker Compose (`docker compose up` starts db, backend, frontend)
- Why: Go gives a simple, fast, strongly typed API with explicit SQL and transactions, which fits stock reservation and balance math. React is the UI stack the team knows best. PostgreSQL has real transactions and row locking, needed to prevent double reservation and negative stock. Docker Compose gives one documented command on a fresh clone.
- Considered: Node + React (one language, but Go gives a typed, explicit backend), Next.js full-stack (less setup, but blurs the frontend/backend split and review ownership), SQLite (no setup, weak under parallel writes), MySQL (works, but Postgres is the better fit in the Go ecosystem).

## D2 — Auth approach
## D3 — API contract format
## D4 — Branching & merge rules
