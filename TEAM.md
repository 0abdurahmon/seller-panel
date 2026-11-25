# Team

| Name | GitHub | Strongest at | Wants to learn | Owns (features) |
|------|--------|--------------|----------------|-----------------|
| Nuriddin | @Nuriddin1999 | React, UI, PostgreSQL, Python | Go, backend, deep learning | **Stock**: `PATCH /products/{id}/stock` with a guard against stale edits, the PostgreSQL store, and reserve, release and commit of stock for orders<br>**Balance**: totals and paginated history, the `balance_entries` table and its endpoints<br>**Dashboard**: the summary endpoint (today, last 7 and 30 days, top products) with a business time zone<br>**Orders tables**: the migration for orders, items and status history<br>**Seed**: the `seed` command with a demo shop, 15 products and 30 orders |
| Akbar | @iskanderovv | React, Next.js, PostgreSQL, CI/CD | Go | **Products**: model, validation, endpoints, the category tree, archive, search and filters, up to 5 photos saved with the product<br>**Images**: converting and storing images as WebP (`internal/upload`), used by product photos and the shop logo<br>**Infra**: Docker Compose, hardened images, nginx and rate limits, CI, smoke and end-to-end tests<br>**API docs**: the OpenAPI spec, Swagger UI at `/docs`, the spec as JSON at `/swagger`<br>**Backend conventions**: one shape for every feature, `AGENTS.md`, the rules for database tests |
| Samandar | @01110011r | Distributed systems | Go | **Auth**: sellers and shops in PostgreSQL, email and password sign-up and login with JWT, sign-in by a one-time code sent through Telegram<br>**Shop profile**: `GET` and `PUT /shop` (name, description, phone, pickup address; the logo is a file in `PUT /shop`)<br>**Orders**: the domain, with the status machine, stock reservation and release, and the sale and commission entries on delivery |
| Abdukhalil | @uicodee | Everything | Go | **All frontend** (`frontend/`)<br>**Foundation**: aliases, Tailwind, the folder layout, shared UI components on shadcn<br>**App**: navigation and layout, sign-in and register on the real API<br>**Products screens**: the list with filters, details, create, edit, archive and activate, with photos |

Orders is owned by Samandar. Akbar built the PostgreSQL side and the routes (#97, #99) and Nuriddin the tables and the stock effects orders call (#93, #76), so ask them about those parts, but changes go through Samandar.

The frontend (`frontend/`) is Abdukhalil's. Everyone else owns the backend of their features and agrees the API with him, so the screens of Stock, Balance, Dashboard and the rest are his, built on the endpoints their owners ship.

## Working agreement
- Core hours: 10:00–18:00 (Tashkent, UTC+5)
- Where we talk (chat / call): Telegram group
- PR review response time: within 1 hour (during core hours)
- Merge captain rotation: Day 1 @iskanderovv, Day 2 @Nuriddin1999, Day 3 @01110011r, Day 4 @iskanderovv, Day 5 @Nuriddin1999

## Progress
Merged work per owner, so the team can see what is done without reading the history. Update it in the PR that merges a feature.

**Auth and shop profile (Samandar)**
- Email and password sign-up and login with JWT, sellers and shops in PostgreSQL (#40, #41).
- Shop profile: `GET` and `PUT /shop` (#65). The logo became a file in `PUT /shop` (#121, Akbar).
- Sign-in by a one-time code sent to the seller's Telegram chat (#108, issue #107): `POST /auth/telegram/link`, `POST /auth/request-code`, `POST /auth/verify`. Needs `TELEGRAM_BOT_TOKEN` and `TELEGRAM_BOT_USERNAME`; without them the code endpoints answer `503`. Decision D10 in `docs/DECISIONS.md`.
- Repair: #108 merged with a broken `internal/app/app.go`, so `main` did not compile. Fixed in #120.

**Orders (Samandar)**
- Order domain: status machine, stock reservation and release, sale and commission entries on delivery (#71).
- The PostgreSQL side and routes were built by Akbar (#97, #99).

**Open, not decided**
- Real-time order changes: SSE or polling, with the choice still to make. README lists it as in scope (`README.md`, MVP list).
- Telegram alerts for new orders: not started.
- Frontend sign-in against the Telegram code flow: not started (Abdukhalil).
- Running more than one backend instance: blocked by the bot polling in D10.
