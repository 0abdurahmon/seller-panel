# Rate limits

Every limit in the stack, where it is enforced, what the caller gets when it is hit, and the environment variable that changes it. Defaults are the values in `backend/internal/config/config.go`, and `.env.example` lists them all.

## Overview

| Layer | Limits | Keyed by | Configured by |
|-------|--------|----------|---------------|
| nginx (`frontend/nginx.conf`) | `/api/` 10 requests/s, burst 30; `/api/auth/` 10/min, burst 5; uploads 50/s | client IP | edit the file, not env (see below) |
| Backend, protected routes | requests per second and burst | shop, from the token | `RATE_LIMIT_SHOP_RPS`, `RATE_LIMIT_SHOP_BURST` |
| Backend, images | saving an image per minute | shop | `RATE_LIMIT_IMAGES_PER_MINUTE` |
| Backend, login, code request, code check | failed attempts in a window | client IP and email | `RATE_LIMIT_LOGIN_ATTEMPTS`, `RATE_LIMIT_LOGIN_WINDOW` |
| Backend, sign-in codes | time between codes, wrong tries per code | seller | `RATE_LIMIT_CODE_COOLDOWN`, `RATE_LIMIT_CODE_ATTEMPTS` |
| Backend, order event streams | open streams | shop | `RATE_LIMIT_STREAMS_PER_SHOP` |

The layers stack: a request must pass nginx, then the shop limit, then the route's own limit.

## Settings

| Variable | Default | Meaning |
|----------|---------|---------|
| `RATE_LIMIT_SHOP_RPS` | `20` | Requests per second a shop may make on protected routes. Must be above 0 |
| `RATE_LIMIT_SHOP_BURST` | `40` | Requests a shop may make at once before the rate applies. At least 1 |
| `RATE_LIMIT_IMAGES_PER_MINUTE` | `20` | Requests per minute and shop that save an image (multipart `POST /products`, `PUT /products/{id}`, `PUT /shop`). At least 1 |
| `RATE_LIMIT_LOGIN_ATTEMPTS` | `5` | Attempts per IP and email in the window. The bucket refills at `attempts / window` |
| `RATE_LIMIT_LOGIN_WINDOW` | `15m` | The window, as a Go duration (`90s`, `15m`, `1h`). Must be above 0 |
| `RATE_LIMIT_CODE_COOLDOWN` | `60s` | Minimum time between two sign-in codes for one seller. `0s` removes the wait. Must not be negative |
| `RATE_LIMIT_CODE_ATTEMPTS` | `5` | Wrong codes allowed for one sign-in code before it stops working. At least 1 |
| `RATE_LIMIT_STREAMS_PER_SHOP` | `5` | Open `GET /orders/events` streams per shop. At least 1 |

A value that is not a number or a duration stops the backend at start with a message that names the variable. A value out of range (for example `0` for attempts) stops it too, with a general "rate limit settings must be positive" message.

## What the caller sees

| Limit hit | Response | Body `code` |
|-----------|----------|-------------|
| nginx `/api/` or `/api/auth/` | `429` | none (plain nginx response, no JSON body) |
| shop rate, login, code request, code check, image save, event stream | `429` with `Retry-After` | `rate_limited` |
| sign-in code asked again during the cooldown | `202` with `{"status":"sent"}`, no new message is sent | none |
| wrong sign-in code past the limit, or an expired code | `401` | `invalid_code` |

The cooldown answers `202` on purpose, so a caller cannot tell from the response that a code was recently sent.

## Where each limit applies

- **Shop rate** (`backend/internal/httpapi/router.go`, `protected(...)`): every protected route, per shop. Public routes (`/auth/register`, `/auth/login`, `/auth/request-code`, `/auth/verify`) do not use it.
- **Image rate**: only requests that save an image. A JSON edit does not count.
- **Login guard** (`backend/internal/auth/handler.go`): one guard for `POST /auth/login`, `POST /auth/request-code` and `POST /auth/verify`. The key includes the IP, so a stranger cannot lock a seller out from another network.
  - Login counts only failed attempts. A successful login or code check clears its key.
  - Code requests count every call, successful or not, because each call can send a Telegram message. Five code requests within the window stop the next one, even if all five were sent.
- **Code cooldown and wrong tries** (`backend/internal/auth/service.go`): one code per seller at a time. A new code replaces the old one and starts with 0 wrong tries, so the cooldown is the only thing that limits how often codes are sent.
- **Event streams** (`backend/internal/realtime`): a stream that would be the sixth for a shop gets `429` and `Retry-After: 5`. Streams close when the token expires.

## nginx limits

nginx limits are not read from the environment. `frontend/nginx.conf` holds them:

```
limit_req_zone $binary_remote_addr zone=api_global:10m rate=10r/s;
limit_req_zone $auth_limit_key zone=api_auth:10m rate=10r/m;
```

To change them, edit the file and rebuild the frontend image: `docker compose build frontend && docker compose up -d --no-deps frontend`. Making them environment-driven needs an nginx template, which this repository does not have yet.

Because nginx sees the client IP as `$binary_remote_addr`, every user behind one NAT shares the per-IP limit. Keep it in mind when tuning the `/api/auth/` rate.

## Tuning examples

- Busy shop on a fast connection: `RATE_LIMIT_SHOP_RPS=50`, `RATE_LIMIT_SHOP_BURST=100`.
- Stricter sign-in codes: `RATE_LIMIT_CODE_COOLDOWN=5m`, `RATE_LIMIT_CODE_ATTEMPTS=3`.
- Several staff at the same time on the orders screen: `RATE_LIMIT_STREAMS_PER_SHOP=10`. Each open browser window holds one stream, and a hidden tab holds none.

Changes apply on the next start of the backend (`docker compose up -d --no-deps backend`). The buckets for the shop, image, login and stream limits are in memory, so a restart empties them. The code cooldown is kept in the database, so a restart does not allow an early code.

## Not limited

- The order alert queue (`orderAlertQueue` in `backend/internal/app/app.go`, 100 messages) is a bound on memory, not a rate limit. When it is full, new alerts are dropped and logged (D19).
- The Telegram send has no retry and no rate limit of its own. Telegram's Bot API limits apply.

See also `docs/DECISIONS.md` D6 (shop rate), D10 (sign-in codes) and D18 (order events).
