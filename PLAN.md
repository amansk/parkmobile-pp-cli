# ParkMobile consumer API plan

Hand-maintained reverse-engineering notes for **parkmobile-pp-cli**.

> **Not the partner API.** [developer.parkmobile.io](https://developer.parkmobile.io) is commercial/partner-only (enforcement, gated parking, locations for operators). This CLI targets the **Phonixx consumer ServiceStack API** used by the ParkMobile US mobile/web zone-parking flow.

> **Not ParkNYC / Flowbird.** ParkNYC (`my.nyc.flowbirdapp.com`) is a different product and API surface.

## API host (verified Sep 2026)

| Item | Value | Status |
|------|-------|--------|
| Base URL | `https://parkmobile.us/ParkmobileApi` | **Verified** — public ServiceStack metadata at `/ParkmobileApi/csv/metadata` |
| Stack | ServiceStack 3.x (CSV + JSON) | **Verified** — `x-powered-by: ServiceStack/3.971` on responses |
| CORS headers | `Authorization`, `PMAuthenticationToken`, `SourceAppKey` | **Verified** — from `access-control-allow-headers` |
| Public ping | `GET /locations` → 200 JSON without auth | **Verified** live Sep 2026 |

## Auth (partially verified)

Consumer sessions appear to use a combination of:

1. **Browser cookies** from [parkmobile.io/login](https://parkmobile.io/login) → Zone Parking → [dlweb.parkmobile.us/Phonixx/](https://dlweb.parkmobile.us/Phonixx/) (ASP.NET WebForms; cookies must be seeded from parkmobile.io first — see [menubar.io writeup](https://menubar.io/creating-a-bot-to-refill-parking-meters-using-aws-lambda)).
2. **`PMAuthenticationToken` header** and/or **`Authorization: Bearer …`** (mobile app pattern; exact mapping from web cookies **unverified** without live HAR).
3. **`SourceAppKey` header** (mobile app; value **unverified**).

CLI auth path:

- `auth login --chrome` or `--cookies-file` (Playwright storage-state / Netscape-style raw Cookie header)
- Saved to `~/.config/parkmobile-pp-cli/cookies.json` mode **0600**
- Env override: `PARKMOBILE_COOKIES`

Token endpoint (metadata only, not used by CLI v0):

- `POST /token2/` — `AuthRequest2` (AppId, AppSecret, AppVersion, Scope, Token) — **unverified** for consumer web; likely mobile bootstrap.

## Verified REST routes (public metadata)

Source: `https://parkmobile.us/ParkmobileApi/csv/metadata?op=<OperationName>`

| Operation | Method | Path | CLI use |
|-----------|--------|------|---------|
| (locations) | GET | `/locations` | `doctor --live` ping |
| Identify2 | GET | `/account/identify2` | `account me`, session probe |
| (vehicles) | GET | `/account/vehicles` | `vehicles list` |
| (payment methods) | GET | `/account/paymentmethods` | `account payment-methods` |
| ParkingZoneInfoRequestV4 | GET | `/v4/parking/zone/{ZoneCode}` | `zones get` (primary) |
| ParkingZoneInfoRequest | GET | `/v3/parking/zone/{ZoneCode}` | `zones get` fallback |
| ParkingZone | GET | `/parking/zones/{SignageCode}`, `/parking/zone/{InternalZoneCode}` | `zones get` fallback |
| ParkingPriceInfoRequest | GET | `/v3/parking/price` | `session start preview`, inline quote |
| Parkingactions / ParkingActionsHistory | GET | `/v2/parking/history`, `/parking/history` | `sessions list` |
| ParkingActivateRequest | POST | `/v3/parking/active` | `session start` (**body unverified**) |
| ParkingExtensionActivateRequest | PUT | `/v3/extension/active` | `session extend` (**body unverified**) |
| ParkingStop | DELETE | `/parking/active/{Id}` | `session stop` (**unverified** live) |

## Unverified / guessed

| Item | Notes |
|------|-------|
| `GET /v3/parking/price` query params | CLI sends `zoneCode`, `durationInMinutes`, optional `orderToken` — inferred from CSV `ParkingPriceInfoRequest` (`order_token`, `duration_in_minutes`) |
| `POST /v3/parking/active` JSON body | CLI sends `zoneCode`, `durationInMinutes`, optional `vehicleId`, `billingMethodId`, `spaceNumber`, `orderToken` — **needs live HAR** |
| `PUT /v3/extension/active` JSON body | CLI sends `parkingActionId`, `durationInMinutes`, optional `billingMethodId`, `timeblockId` — **needs live HAR** |
| `SourceAppKey` value | Not documented publicly |
| Nearby zone search | No verified lat/lon search on Phonixx US API in metadata (Hackatrain 2019 sandbox had `/inventory/GetLocationByLatLon/` on a **different host** — not production US) |
| parkmobile.io Next.js app | May call newer BFF endpoints not yet mapped — **not wired in v0** |

## Hackatrain 2019 sandbox (not production)

`https://hackatrain.parknowportal.com/` — historical EasyPark-group sandbox (`x-api-key` auth). Useful shape reference only; **do not treat as live US consumer API**.

## Hard gates (CLI)

| Action | Confirm string |
|--------|----------------|
| Start | `START PARKMOBILE SESSION` |
| Extend | `EXTEND PARKMOBILE SESSION` |
| Stop | `STOP PARKMOBILE SESSION` |

All require `--enable-live-parking --owner-approved --confirm "<phrase>"`.

## Smoke test (Amandeep)

1. Sign in at [parkmobile.io/login](https://parkmobile.io/login) → **Zone Parking** in Chrome.
2. `parkmobile-pp-cli auth login --chrome`
3. `parkmobile-pp-cli doctor --live --json`
4. Read-only: `account me`, `vehicles list`, `zones get --zone <code>`, `sessions list`
5. Quote: `session start preview --zone <code> --duration-minutes 60 --json`
6. Capture HAR during a manual start/extend/stop in the app to verify mutation bodies before relying on live gates.

## Publishing

Target path in [printing-press-library](https://github.com/mvanhorn/printing-press-library): `library/travel/parkmobile/` or `library/commerce/parkmobile/`.
