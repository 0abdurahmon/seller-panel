# Seller Panel — Team Onboarding Project

Welcome aboard! 👋

For the next **3–5 days** you will work **as one team** to build a **Seller Panel**: the web app a seller on an online marketplace uses to run their shop — list products, manage stock, handle orders, and see how much money they have made.

You choose the stack. You split the work. You run the process.

> **What we care about most is _how_ you work together, not only _what_ you ship.**
> A smaller panel built through a clean, visible team workflow beats a big panel that one person wrote alone at 3 a.m.

---

## 1. The product

A marketplace connects **sellers** with **buyers**. Buyers browse and order. Sellers need a panel to keep their shop running. You are building that panel.

### Must have (MVP)

| # | Area | What the seller can do |
|---|------|------------------------|
| 1 | **Auth** | Sign up / log in (phone + one‑time code, or email + password — your choice). Log out. Only see their own data. |
| 2 | **Shop profile** | Set shop name, logo, description, contact phone, pickup address. |
| 3 | **Products** | Create, edit, archive a product: title, description, category, price, photos (at least 1), stock quantity, SKU. List with search, filter by category/status, pagination. |
| 4 | **Stock** | Change stock quickly from the list. A product with 0 stock shows as *Out of stock*. Low‑stock warning (threshold you decide). |
| 5 | **Orders** | See incoming orders, open an order, move it through statuses: `new → accepted → packed → handed_to_courier → delivered`, or `cancelled` (with a reason). Accepting an order reserves stock; cancelling returns it. |
| 6 | **Dashboard** | Home page with: today's orders, orders waiting for action, revenue for the last 7 / 30 days, top 5 products. |
| 7 | **Balance** | Show earned money: sum of delivered orders minus a platform commission (e.g. 10%), plus a simple history table. |

### Nice to have (only after MVP is solid)

- Order status changes in real time (WebSocket / SSE / polling)
- Uzbek / Russian / English language switch
- Bulk product import from CSV/Excel
- Product variants (size / color) with their own stock
- Notifications (new order, low stock)
- Mobile‑friendly layout
- Dark mode

### Out of scope

The buyer app, real payments, real couriers. **Fake what is outside the panel** — a seed script or a small "simulate buyer order" button is enough to create orders.

See [`docs/DOMAIN.md`](docs/DOMAIN.md) for entities, statuses and rules.

---

## 2. The rules of the game

### Stack — your choice
Frontend, backend, database, hosting — **you decide as a team**. Write the decision and the *why* in [`docs/DECISIONS.md`](docs/DECISIONS.md) on Day 1. Pick what the team can actually deliver in 3–5 days, not what sounds coolest.

A backend is expected (not only `localStorage`). It can be as simple as you like.

### Workflow — this is the main part
We want to see a real team workflow on GitHub, the same way we work every day:

1. **Plan in Issues.** Break the work into small issues (half a day or less each). Use the issue template. Every issue has an owner.
2. **Track on a Project board.** Create a GitHub Project (`Backlog → In progress → In review → Done`). The board must reflect reality at any moment.
3. **Branch per issue.** `feat/12-product-list`, `fix/31-stock-negative`, … Never commit directly to `main`.
4. **Pull Requests for everything.** Small PRs (aim < 400 changed lines). Use the PR template. Link the issue (`Closes #12`). Add a screenshot or short video for any UI change.
5. **Review each other.** Every PR needs **at least 1 approval from a teammate** before merge. Reviews should contain real comments, not just "LGTM".
6. **Keep `main` green.** Set up CI (GitHub Actions) on Day 1: lint + build + tests on every PR. Don't merge red PRs.
7. **Daily stand‑up note.** Each day, append a short entry to [`docs/STANDUP.md`](docs/STANDUP.md): what each person did, will do, and what blocks them.
8. **Conventional commits** are welcome (`feat:`, `fix:`, `chore:` …).

### Balance the team
- Fill in [`TEAM.md`](TEAM.md) on Day 1: who you are, what you're strongest at, and what you want to learn.
- Split ownership **by feature, not by layer** where possible (e.g. one person owns *Orders* end‑to‑end), so everyone touches frontend, backend and data.
- Pair up when someone is stuck for more than ~1 hour. Ask early.
- Everyone must have **merged PRs and done reviews**. We will look at the contribution graph — it should not be one person.
- Pick a rotating **"merge captain"** each day who watches the board, unblocks people and keeps `main` releasable.

---

## 3. Timeline

| Day | Goal |
|-----|------|
| **Day 1** | Kick‑off. Fill `TEAM.md`. Choose stack → `DECISIONS.md`. Design the data model & API contract. Create issues + board. Repo skeleton, CI, one "hello world" deploy or a working `docker compose up`. |
| **Day 2** | Auth, shop profile, products CRUD. First PRs reviewed and merged. |
| **Day 3** | Stock, orders flow, seed / fake‑order generator. **Mid‑point demo** (15 min): show what works, what's at risk. Cut scope if needed. |
| **Day 4** | Dashboard, balance, polish, tests for the important rules (stock reservation, status transitions, commission math). |
| **Day 5** | Bug bash, README run instructions, final demo + short retro. |

On a **3‑day** schedule: Day 1 = Day 1, Day 2 = Days 2–3, Day 3 = Days 4–5. Cut the nice‑to‑haves and keep the MVP and the workflow.

---

## 4. Definition of Done

A feature is *done* when:
- it's merged to `main` through a reviewed PR with green CI,
- the related issue is closed and the board is updated,
- it works on a fresh clone following the README run steps,
- important business rules have at least one test.

The project is *done* when:
- `main` runs with **one documented command** (e.g. `docker compose up`, or `npm i && npm run dev`) and seed data,
- (bonus) it is deployed somewhere and the link is in this README,
- you present a **10–15 min demo** as a team: everyone speaks about the part they built,
- you hold a **short retro** and write it to `docs/RETRO.md` (what went well, what didn't, what you'd change).

---

## 5. How we evaluate

See [`docs/EVALUATION.md`](docs/EVALUATION.md). In short:

| Weight | Area |
|-------:|------|
| **35%** | Team workflow — issues, board, PRs, reviews, CI, stand‑ups, fair split |
| **30%** | Product — MVP works end‑to‑end, business rules are correct |
| **20%** | Code quality — structure, naming, tests, no secrets in git |
| **15%** | Communication — decisions written down, demo, retro |

---

## 6. Getting started

```bash
git clone https://github.com/0abdurahmon/seller-panel.git
cd seller-panel
```

You will be added as collaborators. Branch protection on `main` requires a PR + 1 approval.

## How to run

Stack: Go (backend), React + TypeScript + Vite (frontend), PostgreSQL. See [`docs/DECISIONS.md`](docs/DECISIONS.md).

### Prerequisites
- Docker with Compose v2 (for the one-command run)
- For local development without Docker: Go 1.26+, Node 20.19+ (or 22.12+) and npm

### One command (Docker)

```bash
docker compose up --build
```

| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| Backend | http://localhost:8080 (`GET /health`, this machine only) |
| PostgreSQL | localhost:5432 (this machine only) |

The frontend page shows `Backend: ok` when everything is connected. Every service has a healthcheck: `docker compose ps` shows `healthy`, and `docker compose up --build --wait` returns only when all three are healthy. The backend and the database are published on `127.0.0.1` only; the frontend is the single entry point for other machines. The apps run as non-root users with a read-only filesystem and no extra Linux capabilities. Stop with `Ctrl+C`; remove containers and the database volume with `docker compose down -v`.

### Configuration

Every variable has a default, so no `.env` file is needed to start. To override, copy the example and edit it:

```bash
cp .env.example .env
```

| Variable | Default | Meaning |
|----------|---------|---------|
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | `seller`, `change-me`, `seller` | Database credentials |
| `POSTGRES_PORT`, `BACKEND_PORT`, `FRONTEND_PORT` | `5432`, `8080`, `3000` | Ports on your machine (change if one is taken) |
| `JWT_SECRET` | `change-me-to-a-long-random-string` | Token signing key, always change outside local use |
| `JWT_TTL` | `24h` | How long a login token stays valid (Go duration, for example `12h`) |
| `COMMISSION_PERCENT` | `10` | Platform commission on delivered orders |
| `LOW_STOCK_THRESHOLD` | `5` | Stock level that shows the low-stock warning |

Never commit `.env`. Only `.env.example` is tracked.

### Local development (without Docker for the apps)

```bash
docker compose up db
cd backend && go run ./cmd/server
cd frontend && npm install && npm run dev
```

The Vite dev server runs on http://localhost:5173 and proxies `/api` to the backend (`VITE_API_URL`, default `http://localhost:8080`).

### Checks (same as CI)

```bash
cd backend && gofmt -l . && go vet ./... && go test ./... && go build ./...
cd frontend && npm run lint && npm run build
```

### Seed data

Not available yet. The seed script and the dev "simulate order" endpoint come with the database and orders issues; this section will be updated then.

## Live demo

_Link here (optional)._

---

Questions about the task? Open an issue with the `question` label and tag your mentor. Good luck — have fun building it together! 🚀
