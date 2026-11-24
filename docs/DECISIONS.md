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
- Email + password. `POST /auth/register` creates the seller and their shop in one transaction and returns a token; `POST /auth/login` returns a token. Password 8–72 bytes (bcrypt reads only 72), stored as a bcrypt hash; email trimmed and lower-cased; shop name 1–120 characters.
- Token: JWT signed with HS256 and `JWT_SECRET`, lifetime `JWT_TTL` (default 24h), carrying `seller_id` and `shop_id`. Sent as `Authorization: Bearer <token>`; only HS256 is accepted, expired or foreign tokens get `401 unauthorized`.
- `auth.RequireAuth` checks the token and puts the principal in the request context; handlers read the shop only from there (`auth.MustPrincipal`), never from the body or URL. Every repository filters by that `shop_id`, so another seller's data is `404`.
- Wrong password and unknown email both answer `401 invalid_credentials` and take about the same time (a dummy bcrypt compare), so the API does not reveal which emails exist. A taken email is `409 email_taken`.
- Logout is client-side: the frontend drops the token. No server session store. The frontend will keep the token in `localStorage` (ARCHITECTURE open question 1, a known trade-off).
- Why: works without any outside service, is easy to test, and fits a 3–5 day MVP; one token per seller already scopes every request to one shop.
- Considered: phone + one-time code (needs an SMS provider and rate limiting), server sessions with cookies (state on the server, CSRF handling), refresh tokens (more moving parts than an MVP needs; a seller logs in again after 24h).

## D3 — API contract format
- The contract is OpenAPI 3.0 in `api/openapi.yaml`. It is the single source of truth for paths, request and response schemas and error cases; field names equal the JSON tags in Go.
- Every new or changed endpoint is described in the same PR as its code (AGENTS.md). Swagger UI at `/docs/` (behind basic auth from `SWAGGER_USER` / `SWAGGER_PASSWORD`) shows the file; `npx @redocly/cli lint api/openapi.yaml` validates it.
- Shared conventions live in the file once: Bearer auth, `snake_case` JSON, money as integer so'm, `?page=&page_size=` lists returning `{items, page, page_size, total}`, and errors as `{code, message}` with the codes from AGENTS.md.
- The frontend will generate its API types from this file (ARCHITECTURE section 7), so the client cannot drift from the backend.
- Why: frontend and backend can work in parallel against one agreed contract, reviewers see API changes as a diff, and the docs page comes for free.
- Considered: a shared TypeScript types package (no docs page, Go side still hand-written), a markdown table in the README (drifts silently, no validation), GraphQL / tRPC (new tooling for the team, tRPC needs a TypeScript backend).

## D4 — Branching & merge rules
- Never commit directly to `main`. Every change is an issue, a branch and a pull request.
- Branch name: `<type>/<issue-number>-<short-name>`, for example `feat/12-product-list`, `fix/31-stock-negative`, `docs/4-branching-rules`. Types: `feat`, `fix`, `docs`, `chore`, `test`, `refactor`.
- Commits follow Conventional Commits: `feat:`, `fix:`, `docs:`, `chore:`, `test:`, `refactor:`.
- Pull requests: one issue each, aim for fewer than 400 changed lines, use the PR template, link the issue with `Closes #N`, add a screenshot or short video for any UI change.
- Review: at least 1 approval from a teammate, never the author. Comments must be specific, not only "LGTM". The author fixes comments on the same branch.
- Merge: only when CI is green and the PR is approved. Use squash and merge, then delete the branch. The author merges their own approved PR.
- Merge captain: one person per day, rotating, listed in `TEAM.md`. Watches the board, unblocks reviews and keeps `main` releasable.
- Stuck for more than about 1 hour: ask a teammate or pair up.
- Why: small reviewed PRs keep `main` working for three people changing the same repo, and the history shows who did what.
- Considered: trunk-based direct commits (faster, no review trail), long-lived feature branches (big merges and conflicts).

## D5 — Stock
- Own package `backend/internal/stock`, separate from `product`, because Stock and Products have different owners (`TEAM.md`). It writes only `stock` and `updated_at` of the `products` table and reads `reserved` and `status`; the table itself comes from the products migration.
- `available = stock − reserved`. Label: `out_of_stock` when available is 0, `low_stock` when 0 < available ≤ threshold, `in_stock` otherwise. The backend returns the label (`stock_label`), so every screen shows the same one.
- Low-stock threshold: one global value, `LOW_STOCK_THRESHOLD` (default 5, `0` turns the warning off). Not per shop in the MVP.
- Manual change: `PATCH /products/{id}/stock` with `{"stock": 12, "expected_stock": 10}`, behind auth. `stock` is the new absolute quantity, not a delta, so a retried request gives the same result. Rules: integer 0–1 000 000; not below `reserved` (`409 stock_below_reserved`, the seller cannot take back goods promised to accepted orders); archived products cannot change (`409 product_archived`); another shop's product is `404 not_found`. Response: `{product_id, stock, reserved, available, stock_label, updated_at}`.
- `expected_stock` (required) is the stock the seller saw before editing. If it no longer matches (another tab, an order), nothing is written and the answer is `409 stock_changed`; the UI reloads the product and shows the real number. Error bodies are the shared `{code, message}` (AGENTS.md).
- The rules are in one `UPDATE … WHERE` statement, so a concurrent order accept or a second tab cannot slip between a check and the write.
- Orders declare a small interface that `stock.PgRepository` implements (ARCHITECTURE rule 4): `Reserve` (accept), `Release` (cancel) and `Commit` (deliver), passing their transaction (`database.DBTX`), one item at a time in ascending product id to avoid deadlocks. `Reserve` cannot oversell (`stock − reserved >= qty` in the `WHERE`) and returns `ErrInsufficientStock`; `Release`/`Commit` return `ErrReservationMismatch` if fewer units are reserved than the order holds, so the order transaction rolls back.
- Why: one owner per feature, one place for stock math, no silent lost updates, and the database `CHECK` constraints stay the last line of defence.
- Considered: stock inside `product` (one package, but two owners editing it), delta updates `{"change": -2}` (shorter for +/- buttons, but a retried request would apply twice), last write wins without `expected_stock` (simpler, but one tab silently overwrites another), per-shop threshold (more settings UI for little MVP value).

## D6 — Rate limiting
- Four layers, no Redis. Nginx (shared by every client of the stack): `/api/` limited to 10 requests per second per IP with a burst of 30, and `/api/auth/` to 10 per minute per IP with a burst of 5. Backend: `httpx.RateLimit`, a token bucket per key, for protected routes per shop (`RATE_LIMIT_SHOP_RPS` 20, `RATE_LIMIT_SHOP_BURST` 40), plus a login guard.
- Login guard: 5 failed attempts per IP and email in 15 minutes (`RATE_LIMIT_LOGIN_ATTEMPTS`, `RATE_LIMIT_LOGIN_WINDOW`), then `429` until a token refills (one per 3 minutes). Only failed attempts count and a successful login resets the counter. While blocked, even the right password is refused, so guessing cannot win on attempt 6. The key includes the IP so a stranger cannot lock a seller out of their account from another network.
- Answer: `429` with `Retry-After` and `{"code":"rate_limited","message":"..."}`, the same shape from nginx and from the backend.
- Client IP: the backend uses `X-Real-IP` only when the request comes from a trusted proxy (`TRUSTED_PROXIES`, default loopback and private ranges, which covers the nginx container). Any other client cannot spoof its address with a header.
- Limits live in process memory and idle keys are swept every minute. They reset on restart and are counted per backend instance. That is fine for one instance. If the backend is scaled out, move the counters to Redis.
- Why per shop and not per IP for protected routes: sellers behind one office or mobile network share an IP, a shop ID is the real unit of fairness. Per IP stays at the nginx edge, where no token exists yet.
- Why not a rule per endpoint: new endpoints get the global and per shop limits without anyone remembering to add them. Only the two routes with a real abuse case (login, register through `/api/auth/`) get a stricter rule.
- Not included: CAPTCHA, permanent account lock, Redis.

## D7 — Balance
- Read only. The order feature writes the entries when an order becomes `delivered`: `sale` (+total) and `commission` (−total × `COMMISSION_PERCENT` / 100, integer division), in the same transaction as the status change. `payout` exists in the model; the MVP never creates one.
- `GET /balance`: `available` = sum of all entries, `earned` = sum of `sale`, `commission` = sum of `commission` (negative). One `SUM … FILTER` query; a shop with no entries gets zeros, not 404.
- `GET /balance/entries`: newest first (`created_at DESC, id DESC`, so equal times keep a stable order), `?page=&page_size=` clamped like product lists (default 20, max 100), `items` is `[]` when empty.
- Both routes are protected and filter by the shop from the token; another shop's entries are never visible.
- The `balance_entries` table comes from migration `0004_balance_entries` (Balance), in the shape of `backend/docs/schema.dbml`. The foreign key `order_id → orders` arrives with migration `0005_orders`, because the orders table did not exist when Balance was built. The unique key `(order_id, type)` and the sign checks protect the money: one sale and one commission per order, and a payout never has an order.
- Why compute totals on read: entries are append only and few per order, the sum is one indexed query, and there is no stored total that could drift from the history.
- Considered: a stored running balance on `shops` (faster for huge histories, but two places to keep in sync), net per order instead of separate entries (simpler table, but loses the commission line the seller wants to see).

## D8 — Orders
- Package `backend/internal/order`: list, detail and `POST /orders/{id}/transition`, all behind auth and filtered by the shop from the token. Another shop's order is a `404`, like a missing one. Orders are created by the dev endpoint `POST /dev/simulate-order` (only when `APP_ENV=development`; in production the route is not registered, so it is the JSON 404) and later by the seed script, never by the seller. It picks 1 to 3 active products with available stock, a quantity of at most 3 and at most the available units, and a fake buyer; it reserves nothing, accepting does. With no such product the answer is `409 no_products_to_order`.
- A transition is one transaction: `SELECT … FOR UPDATE` on the order row, the status machine from `DOMAIN.md` checked in the service, the stock effects, the balance entries, the history row and the status update. Any error rolls everything back. Two simultaneous requests for the same order cannot both succeed: the second one waits for the lock, sees the new status and answers `409 invalid_transition`.
- Stock effects are single `UPDATE … WHERE` statements on `products` (the stock rules of D5 stay in one place per statement): accept reserves with `stock - reserved >= qty` in the `WHERE`, so two orders cannot oversell; cancel releases only what was reserved (from `accepted` or `packed`); delivery takes the quantity out of `stock` and `reserved`. The items are locked in ascending `product_id` order so two orders that share products cannot deadlock.
- Accepting does not look at the product status: archiving hides a product from new orders but keeps the reservations of existing ones (D5).
- Delivery writes the `sale` entry (+total) and the `commission` entry (−`total × COMMISSION_PERCENT / 100`, integer division, so it rounds down and no so'm is lost) in the same transaction (D7).
- Every status change writes one `order_status_history` row (`from_status`, `to_status`, `reason`, time) in the same transaction. `GET /orders/{id}` returns them oldest first as `history`, so the order page can show a timeline.
- The first history row of a created order goes from `NULL` to `new`, so every order has a complete timeline.
- `Order.number` is a sequence in the table and the string `#100245` in the API. Prices and titles on the items are snapshots from the time of the order.
- Lists: newest first (`created_at DESC, id DESC`), `?status=&page=&page_size=` clamped like product lists (default 20, max 100); a status that does not exist or a value that is not an integer is a `400`.
- Ownership: the domain code came with Samandar's orders PRs (#71, #76). Nothing was in progress on the PostgreSQL side, so issue #70 was taken over to finish it (repository, wiring, tests); the domain code keeps its behaviour.
- Considered: stock effects through the stock package (one more hop, but the order transaction would need the stock package to accept a transaction), optimistic locking with a version column instead of `FOR UPDATE` (no waiting, but retries and a new column), history as a JSON column on `orders` (fewer joins, but no per-row timestamps or constraints).

## D9 — The spec as JSON
- Swagger UI at `/docs/` only shows `api/openapi.yaml`. Type and client generators and Postman want JSON from a URL, so the backend serves it: `GET /openapi.json`, made from the file in `OPENAPI_FILE` on request (cached by modification time and size, so an edit shows at once and nothing is converted twice). A missing file is a JSON 404 `spec_not_found`, an invalid one a 500 that hides the details and logs them. With `OPENAPI_FILE` empty the route does not exist.
- nginx publishes it as `GET /swagger` and `GET /docs/spec/openapi.json` behind the same basic auth as `/docs/` (D2/#50 keep the docs behind a login). It blocks `/api/openapi.json` so the login cannot be skipped through the API prefix. The backend port itself is bound to `127.0.0.1`, so the unauthenticated route is not reachable from the network.
- Docker Compose mounts `./api` read-only into the backend too, because the spec lives outside the Go module and cannot be embedded without moving it (it is shared with the frontend generators and Swagger UI).
- Why in the backend and not in the docs container: it is testable in Go (unit and e2e tests), works with `go run`, and needs no extra binary or start script. Keys come out in alphabetical order, which is cosmetic.
- Considered: converting with `yq` when the frontend container starts (works, but needs a binary in the image, a start script, and the JSON goes stale until a restart), baking the JSON at build time (the image is built from `./frontend` and cannot see `api/`), committing a generated `openapi.json` (two copies that can drift), `go:embed` (needs the spec inside `backend/`).

## D10 — Sign-in with a Telegram code
- A seller links a Telegram chat once: `POST /auth/telegram/link` (signed in) returns a `t.me` link with a one-time token valid 10 minutes. Opening it sends `/start <token>` to the bot, which stores the chat on the seller. The token is stored only as a SHA-256 hash and is deleted when used.
- Sign-in: `POST /auth/request-code` sends a 6-digit code to the linked chat (valid 5 minutes, one code per 60 seconds, stored as a hash). `POST /auth/verify` returns the same `TokenResponse` as password login. A code works once and stops working after 5 wrong attempts. Verification is guarded like login (5 failures per IP and email in 15 minutes).
- Answers do not reveal accounts: `request-code` always returns `202` for a known or unknown email, and sends nothing when there is no linked chat. Without `TELEGRAM_BOT_TOKEN` the two endpoints return `503 telegram_unavailable`, and password login is unaffected.
- The bot gets updates by long polling (`getUpdates`) inside the backend process, so one backend instance must run per bot. Several instances would fight over updates. Switching to a webhook is the way to scale out.
- Why: it reverses the D2 trade-off without an SMS provider. Telegram is already the team's channel, costs nothing, and a bot can deliver codes. Password login stays so a seller who has not linked yet is not locked out.
- Considered: SMS codes (needs a paid provider, and phone numbers must be verified), email codes (needs an SMTP service, and the email is already the login name), TOTP apps (the seller must set up an app, more friction for a shop owner), and a webhook instead of polling (needs a public HTTPS address, which the MVP deployment does not have).
- Trade-off: the code is sent in plain Telegram chat and the seller must keep access to that account. A stolen Telegram account can sign in, which is why password login remains available and a relink replaces the chat.
