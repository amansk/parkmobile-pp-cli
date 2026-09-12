# parkmobile-pp-cli

Agent-native [Printing Press](https://github.com/mvanhorn/cli-printing-press) CLI for **ParkMobile US consumer accounts**. Look up zones, list parking sessions, and **hard-gated** start/extend/stop zone parking — using the Phonixx consumer session API at `parkmobile.us/ParkmobileApi`, **not** the partner [developer.parkmobile.io](https://developer.parkmobile.io) API.

**Author:** [Amandeep Khurana](https://github.com/amansk) (@amansk) · **License:** Apache-2.0

## Install

```bash
go install github.com/amansk/parkmobile-pp-cli/cmd/parkmobile-pp-cli@latest
```

Or build from source:

```bash
git clone https://github.com/amansk/parkmobile-pp-cli
cd parkmobile-pp-cli
go build -o parkmobile-pp-cli ./cmd/parkmobile-pp-cli
```

## Quick start

1. Sign in at [parkmobile.io/login](https://parkmobile.io/login) in Chrome and choose **Zone Parking** (cookies must be seeded from parkmobile.io before the Phonixx session works).
2. Import session cookies:

```bash
parkmobile-pp-cli auth login --chrome
# or: parkmobile-pp-cli auth login --cookies-file ./storage-state.json
```

3. Verify setup:

```bash
parkmobile-pp-cli doctor --json
parkmobile-pp-cli doctor --live --json
parkmobile-pp-cli auth status
parkmobile-pp-cli account me --json
parkmobile-pp-cli vehicles list --json
parkmobile-pp-cli account payment-methods --json
```

Payment methods output is ids + last4 only — never full PAN or session secrets.

4. Zones and sessions (read-only):

```bash
parkmobile-pp-cli zones get --zone 1234 --duration-minutes 60 --json
parkmobile-pp-cli sessions list --json
parkmobile-pp-cli sessions get 99 --json
```

## Session start (hard-gated)

Preview never charges:

```bash
parkmobile-pp-cli session start preview --zone 1234 --duration-minutes 60 --json
```

Live start requires **all three** gates (exact confirm string). Mutation request bodies are scaffolded from public Phonixx metadata and marked **unverified** until confirmed with a live HAR — use `--dry-run` first.

```bash
parkmobile-pp-cli session start --zone 1234 --duration-minutes 60 \
  --vehicle-id 1 --billing-method-id 10 \
  --enable-live-parking --owner-approved \
  --confirm "START PARKMOBILE SESSION" --json
```

Inspect payload without posting:

```bash
parkmobile-pp-cli session start --zone 1234 --duration-minutes 60 \
  --enable-live-parking --owner-approved \
  --confirm "START PARKMOBILE SESSION" --dry-run --json
```

## Extend / stop (hard-gated)

Extend (only when `can_extend` is true on the session):

```bash
parkmobile-pp-cli session extend --session-id 99 --duration-minutes 30 \
  --enable-live-parking --owner-approved \
  --confirm "EXTEND PARKMOBILE SESSION" --json
```

Stop early (only when `can_stop` is true):

```bash
parkmobile-pp-cli session stop --session-id 99 \
  --enable-live-parking --owner-approved \
  --confirm "STOP PARKMOBILE SESSION" --json
```

## Global flags

| Flag | Description |
|------|-------------|
| `--json` | Machine-readable JSON output |
| `--agent` | `--json --no-color --no-input` (does **not** imply `--yes`) |
| `--dry-run` | Skip mutating POST/PUT/DELETE where supported |
| `--home` | Override config dir (`$PARKMOBILE_PP_HOME` or `~/.config/parkmobile-pp-cli`) |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage / validation |
| 3 | Not found |
| 4 | Auth |
| 5 | API error |
| 7 | Rate limit / transient |

## Auth details

- Cookies saved to `~/.config/parkmobile-pp-cli/cookies.json` with mode **0600**.
- Override with `PARKMOBILE_COOKIES` (raw Cookie header).
- **`auth status` and `doctor` never print secret values.**

## Smoke testing (live credentials)

This repo builds and tests without live ParkMobile credentials (mock HTTP in unit tests). To validate against production:

1. Complete Chrome login flow above.
2. Run `doctor --live`, then read-only commands.
3. Capture a browser HAR while manually starting/stopping a session to verify mutation bodies in [PLAN.md](./PLAN.md).

## Development

```bash
go test ./...
go vet ./...
```

See [PLAN.md](./PLAN.md) for consumer API endpoint notes (verified vs unverified).

## Publishing

Intended for eventual `/printing-press-publish` into [mvanhorn/printing-press-library](https://github.com/mvanhorn/printing-press-library) under `library/travel/parkmobile/` (or `library/commerce/parkmobile/`).
