# Evaluation

Each area is scored 0–5, then weighted.

## Team workflow — 35%
- [ ] Work is planned as small issues with owners; board reflects reality
- [ ] Every change went through a PR linked to an issue
- [ ] Every PR has ≥ 1 teammate review with meaningful comments
- [ ] CI runs on every PR; `main` stayed green
- [ ] Branch naming and commit messages are consistent
- [ ] Daily stand‑up notes exist for every day
- [ ] Contribution is balanced — everyone authored PRs **and** reviewed others
- [ ] Scope was cut consciously when time was short (visible in issues/notes)

## Product — 30%
- [ ] MVP features 1–7 work end‑to‑end on a fresh clone
- [ ] Seller cannot see or change another seller's data
- [ ] Order status machine enforced on the backend
- [ ] Stock reservation / release is correct (no negative stock, no double reservation)
- [ ] Commission & balance math is correct (integer money)
- [ ] UI is usable: loading, empty and error states exist

## Code quality — 20%
- [ ] Clear project structure, readable naming
- [ ] Tests for the critical rules (status transitions, stock, commission)
- [ ] Input validation on the backend
- [ ] No secrets committed (`.env.example` only)
- [ ] One‑command run documented in README

## Communication — 15%
- [ ] `DECISIONS.md` explains stack and key trade‑offs
- [ ] Mid‑point and final demo: everyone presents their part
- [ ] `RETRO.md` is honest and specific
