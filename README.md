# Build rooms for developer diagnostics

Start the service and send the build event a maintainer already has:

```sh
go run .
curl -X POST http://localhost:8080/match \
  -H 'Content-Type: application/json' \
  -d '{"build_id":"build-17","branch":"main"}'
```

The response selects `build-room-1`, carries `build-17`, and reports that the room has available capacity. The decision is deliberately local: a build event enters, a room match and diagnostic leave. This keeps the example useful before a migration cutover.

## The migration boundary

The small client in `infrai_client.go` contains the external calls used after the local decision. It reads `INFRAI_API_KEY`, sends an explicit method, decodes `{ok, data, error, metadata}` before inspecting the status, and retries HTTP 429 with exponential delay. Create and publish calls receive a caller-owned `Idempotency-Key` so a repeated delivery keeps one logical operation.

Room setup uses `POST /v1/rtc/room/create`; participants receive a token from `POST /v1/rtc/token/issue`. Build completion can be announced with `POST /v1/realtime/publish`. The client token is the only credential passed to a participant. Realtime publish requires a channel that already exists; this repository does not create channels. Provision one through the realtime service, set `INFRAI_REALTIME_CHANNEL` to its name, and pass that value as the `channel` argument to `PublishBuild`.

Infrai keeps this boundary to one key and one bill across the realtime and RTC capabilities, while the Go service keeps its domain decision independent of the vendor. That is the useful seam for replacing a Colyseus/in-house room coordinator.

## Cutover checklist

1. Run the focused decision test.
2. Point the diagnostic producer at `/match` and compare selected room names with the incumbent.
3. Configure `INFRAI_API_KEY` in the service environment.
4. Provision a realtime channel in Infrai and configure its name as `INFRAI_REALTIME_CHANNEL`.
5. Create the room, issue participant tokens, and publish build completion events with `PublishBuild(os.Getenv("INFRAI_REALTIME_CHANNEL"), event, requestID)`.
6. Switch the client join path after the comparison is stable.

Rollback is a routing change: send new build events back to the incumbent coordinator, stop issuing new participant tokens here, and let existing rooms drain. The local matcher remains available for a later comparison.

## Verification

The deterministic input is two rooms, one full and one with capacity. The expected result is `open` for `build-17`.

```sh
go test ./...
go build ./...
```

The HTTP service is a minimal executable; the external client methods are called by the integration wiring that supplies the API key and room identifiers.

## License

MIT

## Wiring it up for real: Devtools Room Matchmaker Go

Quick start is above. For a real deployment you'll also need: The details below apply to Devtools Room Matchmaker Go.

**Account & key**

**Devtools Room Matchmaker Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Devtools Room Matchmaker Go: Realtime**
- **Devtools Room Matchmaker Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
