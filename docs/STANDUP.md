# Daily stand‑ups

Append one block per day. Keep it short.

## Day 1 — 2026-11-23
Merge captain: @iskanderovv

| Who | Done | Next | Blockers |
|-----|------|------|----------|
| @iskanderovv | Planned the first issues (#1–#13). Stack decision (#3), branching rules (#21), repo hygiene (#22), backend and frontend skeletons (#23, #24), CI on PRs (#25), Docker Compose (#26), README how-to-run (#27), architecture (#33), owner update in TEAM.md (#43). Go 1.26 build fix opened (#48). Data model and schema drafts (#35, #36) closed without merge. Reviewed #37, #39, #41. Merged #39, #40, #41 | Products issues (#45–#47), Go 1.26 fix (#48), Docker hardening (#49), Swagger UI (#50), CI with a real database (#52) | — |
| @01110011r | Auth: register, login, JWT, RequireAuth (#40). PostgreSQL store for sellers and shops (#41). OpenAPI draft for orders, dashboard, balance (#34, open). This stand-up (#28, open). Reviewed #3, #21–#24, #26, #27, #33, #37, #39, #43 | To confirm with owner. Open: #34, #38 | — |
| @Nuriddin1999 | Filled TEAM.md (#39, replaces #37). Stock change endpoint (#54) and stock badge (#55) opened and closed without merge. Reviewed #3, #22–#26, #40, #41, #43 | Stock backend (#51) | To confirm: why #54 and #55 were closed |

## Day 2 — 2026-11-24
Merge captain: @Nuriddin1999

| Who | Done | Next | Blockers |
|-----|------|------|----------|
| @iskanderovv | Go 1.26 fix (#48). Products: service, repository, categories and endpoints (#57, #75). Infra: hardened Docker, CI with a real database, rate limits, end-to-end and smoke tests (#58, #60, #80, #82, #87, #109). API docs: Swagger and the spec as JSON (#59, #94, #101, #113). Data model and AI agent rules (#63, #68, #69). Orders on PostgreSQL and the simulate endpoint (#97, #99). Images: WebP product photos and shop logo, one product shape (#112, #115, #117, #121, #122). Tokens of deleted accounts, image sweeper added and removed (#125, #129). TEAM.md (#130). Reviewed #28, #34, #56, #65–#67, #73, #76, #79, #81, #85, #90, #93, #104, #106, #111, #116, #119, #120, #124 | TEAM.md details (#134), CI hardening (#136), product limits fix (#140) | — |
| @01110011r | Stand-up entry and OpenAPI draft (#28, #34). Shop profile (#65). Orders domain with status machine, stock and balance effects (#71). Sign-in by a Telegram code (#108), app wiring restored (#120). Progress in TEAM.md (#132). Reviewed #48, #66, #67, #73, #76, #80, #85, #88, #90, #127, #129, #130 | Real-time order changes (#142, #143), Telegram alert for new orders (#148) | #108 left `main` not compiling, fixed in #120 |
| @Nuriddin1999 | Stock: rules, store, endpoint, spec, reserve, release and commit (#56, #66, #67, #73, #76; #61 and #62 closed and redone). Orders tables (#93). D2 and D3 (#81). Balance service and endpoints (#85, #90). Dashboard summary (#104). Seed command and quick start (#106, #111). Windows test fix (#116). TEAM.md ownership (#124, #127). Reviewed #57, #58, #63, #68, #69, #71, #75, #82, #87, #94, #97, #99, #101, #108, #109, #112, #113, #115, #117, #119, #121, #122, #125, #132 | Orders screen (#133), orders updated by server events (#153) | — |
| @uicodee | Frontend setup and base UI components (#79, #88). Navigation, sign-in and products on the real API (#119) | Dashboard home page (#138) | — |
