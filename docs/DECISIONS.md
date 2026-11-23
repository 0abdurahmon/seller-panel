# Decisions

Record every important decision here: what you chose, what else you considered, and why.

## D1 — Stack (Day 1)
- Frontend:
- Backend:
- Database:
- Hosting / run:
- Why:

## D2 — Auth approach
## D3 — API contract format
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
