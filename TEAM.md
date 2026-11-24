# Team

| Name | GitHub | Strongest at | Wants to learn | Owns (features) |
|------|--------|--------------|----------------|-----------------|
| Nuriddin | @Nuriddin1999 | React, UI, PostgreSQL, Python | Go, backend, deep learning | Stock, Balance, Dashboard |
| Akbar | @iskanderovv | React, Next.js, PostgreSQL, CI/CD | Go | **Products**: model, validation, endpoints, the category tree, archive, search and filters, up to 5 photos saved with the product<br>**Images**: converting and storing images as WebP (`internal/upload`), used by product photos and the shop logo<br>**Infra**: Docker Compose, hardened images, nginx and rate limits, CI, smoke and end-to-end tests<br>**API docs**: the OpenAPI spec, Swagger UI at `/docs`, the spec as JSON at `/swagger`<br>**Backend conventions**: one shape for every feature, `AGENTS.md`, the rules for database tests |
| Samandar | @01110011r | Distributed systems | Go | Auth, Shop profile, Orders |
| Abdukhalil | @uicodee | Everything | Go | All frontend |

Orders is owned by Samandar. Akbar built the PostgreSQL side and the routes (#97, #99) and Nuriddin the tables and the stock effects orders call (#93, #76), so ask them about those parts, but changes go through Samandar.

The frontend (`frontend/`) is Abdukhalil's. Everyone else owns the backend of their features and agrees the API with him, so the screens of Stock, Balance, Dashboard and the rest are his, built on the endpoints their owners ship.

## Working agreement
- Core hours: 10:00–18:00 (Tashkent, UTC+5)
- Where we talk (chat / call): Telegram group
- PR review response time: within 1 hour (during core hours)
- Merge captain rotation: Day 1 @iskanderovv, Day 2 @Nuriddin1999, Day 3 @01110011r, Day 4 @iskanderovv, Day 5 @Nuriddin1999
