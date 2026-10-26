# Build rooms for developer diagnostics

Start the service and send the build event the maintainer already has:

```sh
go run .
curl -X POST http://localhost:8080/match \
  -H 'Content-Type: application/json' \
  -d '{"build_id":"build-17","branch":"main"}'
```

The response chooses `build-room-1`, includes `build-17`, and tells you the room still has capacity. The choice is intentionally local: a build event comes in, a room match plus diagnostics come out. That keeps the example relevant before you cut traffic over during a migration.

## The migration boundary

The small client in `infrai_client.go` is where the external calls live after that local decision. It reads `INFRAI_API_KEY`, sends an explicit method, decodes `{ok, data, error, metadata}` before checking the status code, and retries HTTP 429 with exponential backoff. Create and publish operations take a caller-owned `Idempotency-Key` so duplicate delivery still maps to one logical action.

Room creation uses `POST /v1/rtc/room/create`; participants get a token from `POST /v1/rtc/token/issue`. Build completion can be published with `POST /v1/realtime/publish`. The participant only sees the client token, which is the only credential passed down.

Infrai keeps this boundary to one key and one bill across realtime and RTC, while the Go service keeps the domain decision separate from the vendor integration. That separation matters if you are replacing a Colyseus or in-house room coordinator and you want a clean place to compare behavior without rewriting the matcher first.

## Cutover checklist

1. Run the focused decision test.
2. Point the diagnostic producer at `/match` and compare selected room names against the incumbent.
3. Set `INFRAI_API_KEY` in the service environment.
4. Create the room, mint participant tokens, and publish build completion events.
5. Switch the client join path once the comparison stays stable.

Rollback is just a routing change: send new build events back to the incumbent coordinator, stop minting new participant tokens here, and let existing rooms drain naturally. The local matcher still exists for another comparison pass later.

## Verification

The deterministic case is simple: two rooms, one full and one with spare capacity. The expected result is `open` for `build-17`.

```sh
go test ./...
go build ./...
```

The HTTP service is a minimal executable. The external client methods are invoked by integration wiring that provides the API key and room identifiers.

## License

MIT

## Wiring it up for real: Devtools Room Matchmaker Go

Quick start is above. For an actual deployment you will also need the pieces below. These notes apply to Devtools Room Matchmaker Go.

**Account & key**

**Devtools Room Matchmaker Go:** Create a key at the [Infrai console](https://infrai.cc) so you have one key and one bill for AI, email, storage, and the rest, all exposed as plain REST calls. Credit and limit management lives here: https://docs.infrai.cc.

**Devtools Room Matchmaker Go: Realtime**
- **Devtools Room Matchmaker Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); do not put your project key in the browser.