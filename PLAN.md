# ParkMobile consumer API plan

Hand-maintained reverse-engineering notes for **parkmobile-pp-cli**.

> **Not the partner API.** [developer.parkmobile.io](https://developer.parkmobile.io) is commercial/partner-only (enforcement, gated parking, locations for operators). This CLI targets the **Phonixx consumer ServiceStack API** used by the ParkMobile US mobile/web zone-parking flow.

> **Not ParkNYC / Flowbird.** ParkNYC (`my.nyc.flowbirdapp.com`) is a different product and API surface.

## API host (verified Sep 2026)

| Item | Value | Status |
|------|-------|--------|
| Base URL | `https://parkmobile.us/ParkmobileApi` | **Verified** — public ServiceStack metadata at `/ParkmobileApi/metadata` |
| Stack | ServiceStack 3.x (CSV + JSON) | **Verified** — `x-powered-by: ServiceStack/3.971` on responses |
| CORS headers | `Authorization`, `PMAuthenticationToken`, `SourceAppKey` | **Verified** — from `access-control-allow-headers` |
| Public ping | `GET /locations` → 200 JSON without auth | **Verified** live Sep 2026 |
| Auth probe | `GET /account/identify2` without session → **HTTP 410** | **Verified** live Sep 2026 |

## Auth (partially verified)

Consumer sessions appear to use a combination of:

1. **Browser cookies** from [parkmobile.io/login](https://parkmobile.io/login) → **Zone Parking** → [dlweb.parkmobile.us/Phonixx/](https://dlweb.parkmobile.us/Phonixx/) (ASP.NET WebForms; parkmobile.io alone seeds CSRF cookies — see [menubar.io writeup](https://menubar.io/creating-a-bot-to-refill-parking-meters-using-aws-lambda)).
2. **`PMAuthenticationToken` header** from cookie of the same name (typical Phonixx web session after dlweb login). **`Authorization: Bearer`** only when a real bearer cookie exists — do **not** mirror PMA token as Bearer.
3. **`SourceAppKey` header** (mobile app; value **unverified**).

**Hypothesis confirmed (Sep 2026 review):** importing cookies from **app.parkmobile.io alone** often yields **no `PMAuthenticationToken`**; `doctor --live` session probe will fail until dlweb Zone Parking login completes.

CLI auth path:

- `auth login --chrome` (python3+browser_cookie3 recommended) or `--cookies-file` (Playwright storage-state / raw Cookie header)
- Saved to `~/.config/parkmobile-pp-cli/cookies.json` mode **0600**
- Env override: `PARKMOBILE_COOKIES`
- Chrome import **rejects** imports with parkmobile cookies but no `PMAuthenticationToken`

Token endpoint (metadata only, not used by CLI v0):

- `POST /token2/` — `AuthRequest2` (AppId, AppSecret, AppVersion, Scope, Token) — **unverified** for consumer web; likely mobile bootstrap.

## Verified REST routes (public metadata)

Source: `https://parkmobile.us/ParkmobileApi/json/metadata?op=<OperationName>`

| Operation | Method | Path | CLI use |
|-----------|--------|------|---------|
| (locations) | GET | `/locations` | `doctor --live` ping (no auth) |
| Identify2 | GET | `/account/identify2` | `account me`, `doctor --live` session probe |
| (vehicles) | GET | `/account/vehicles` | `vehicles list` |
| (payment methods) | GET | `/account/paymentmethods` | `account payment-methods` |
| ParkingZoneInfoRequestV4 | GET | `/v4/parking/zone/{ZoneCode}` | `zones get` (primary) |
| ParkingZoneInfoRequest | GET | `/v3/parking/zone/{ZoneCode}` | `zones get` fallback |
| ParkingZone | GET | `/parking/zones/{SignageCode}`, `/parking/zone/{InternalZoneCode}` | `zones get` fallback |
| ParkingPriceInfoRequest | GET | `/v3/parking/price` | `session start preview` (requires `order_token`) |
| Parkingactions / ParkingActionsHistory | GET | `/v2/parking/history`, `/parking/history` | `sessions list` |
| ParkingActivateRequest | POST | `/v3/parking/active` | `session start` (**body: order_token + credit_card**) |
| ParkingExtensionActivateRequest | PUT | `/v3/extension/active` | `session extend` (**body: order_token + credit_card**) |
| ParkingStop | DELETE | `/parking/active/{Id}` | `session stop` (**path verified; body unverified**) |

ServiceStack JSON field names use **snake_case** (e.g. `order_token`, `duration_in_minutes`, `billing_method_id`).

## Unverified / missing (needs live HAR)

| Item | Notes |
|------|-------|
| Checkout → `order_token` | Metadata shows activate/extend/price flows keyed on `order_token`, not zone code. How web/mobile obtains `order_token` before POST `/v3/parking/active` is **unknown** without checkout HAR. |
| `POST /v3/parking/active` full body | Metadata example: `{"order_token":"…","credit_card":{"billing_method_id":…}}` — CLI scaffolds this only; live POST blocked unless `--acknowledge-unverified-body`. |
| `PUT /v3/extension/active` full body | Same `order_token` + `credit_card` shape per metadata. |
| `DELETE /parking/active/{Id}` body | Path verified; optional JSON body in metadata (`id`, `stopTimeLocal`, …) **unverified** live. |
| `GET /v3/parking/price` | Metadata fields: `order_token`, `duration_in_minutes`, `time_block_id` — **no zone_code**. Zone-only quotes are not supported by metadata. |
| `SourceAppKey` value | Not documented publicly |
| Nearby zone search | No verified lat/lon search on Phonixx US API in metadata |
| parkmobile.io Next.js app | May call newer BFF endpoints not yet mapped — **not wired in v0** |

## Hackatrain 2019 sandbox (not production)

`https://hackatrain.parknowportal.com/` — historical EasyPark-group sandbox (`x-api-key` auth). Useful shape reference only; **do not treat as live US consumer API**.

## Hard gates (CLI)

| Action | Required flags |
|--------|----------------|
| Start | `--enable-live-parking --owner-approved --confirm "START PARKMOBILE SESSION"` |
| Extend | `--enable-live-parking --owner-approved --confirm "EXTEND PARKMOBILE SESSION"` |
| Stop | `--enable-live-parking --owner-approved --confirm "STOP PARKMOBILE SESSION"` |

All mutations also require **`--order-token`** (start/extend) where metadata demands it.

Live HTTP mutations (non-`--dry-run`) additionally require **`--acknowledge-unverified-body`** until HAR-verified bodies ship.

## Smoke test (Amandeep)

1. Sign in at [parkmobile.io/login](https://parkmobile.io/login) → **Zone Parking** (dlweb.parkmobile.us) in Chrome.
2. `parkmobile-pp-cli auth login --chrome` (or export cookies via `--cookies-file`)
3. `parkmobile-pp-cli doctor --live --json` — expect `live_ping` **and** `live_session` ok
4. Read-only: `account me`, `vehicles list`, `zones get --zone <code>`, `sessions list`
5. Capture HAR during manual checkout to obtain `order_token`, then `session start preview --order-token <tok> --zone <code> --duration-minutes 60 --json`
6. Inspect mutation payload: `session start --dry-run --order-token <tok> …` (all gates + dry-run)
7. Do **not** live-start until HAR confirms body shape; if testing live anyway, add `--acknowledge-unverified-body`

## Publishing

Target path in [printing-press-library](https://github.com/mvanhorn/printing-press-library): `library/travel/parkmobile/` or `library/commerce/parkmobile/`.
