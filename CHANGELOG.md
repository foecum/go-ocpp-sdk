# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The project is
stable from 1.0.0 on and follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.0.0] - 2026-11-25

First stable release. The public API follows semantic versioning from here.

### Added
- Fuzz tests for OCPP-J framing, payload decoding of every 1.6 and 2.0.1 message,
  and `DateTime`, plus a weekly fuzzing workflow.
- Benchmarks for framing, decoding and a CSMS with up to 1,000 simulated stations;
  indicative results in `docs/benchmarks.md`.
- `examples/loadtest`: a CSMS and many simulated stations in one process, for 1.6
  and 2.0.1.
- `docs/api-review.md`: a review of the public API ahead of v1.0.0.

### Changed
- **Breaking:** `ocppj.CallError` is now `ocppj.CallErrorFrame`, and `csms.CSMS.ChargePoints()` is now `ChargePointIDs()` (`csms/v16` and `csms/v201`).
- The `ChargePoint` types of `csms/v16`, `csms/v201`, `chargepoint/v16` and
  `chargepoint/v201` forward their methods explicitly; method sets are the same as
  in v0.1.0.

## [0.2.0] - 2026-11-25

### Added
- OCPP 2.0.1: all 64 messages, one package per functional block (`v201/security`,
  `v201/provisioning`, and so on), shared types in `v201/types`, and
  `v201.NewRegistry`. The sources are generated from the official JSON schemas by
  `internal/gen201` and tested against them.
- `csms/v201` and `chargepoint/v201`: the high-level APIs, with one handler
  interface per functional block.
- `examples/v201/csms` and `examples/v201/chargingstation`, with an end-to-end
  smoke test.
- The documentation site and the README cover OCPP 2.0.1.

### Fixed
- `ws.Server.Shutdown` no longer waits for its whole context when a connection
  finishes its handshake while the server is shutting down.

### Changed
- `ocpp.DateTime` is the single date-time type for every version (`v16/types.DateTime`
  is an alias).

## [0.1.0] - 2026-11-25

### Added
- OCPP-J framing and the call tracker in `ocppj`: CALL, CALLRESULT and CALLERROR,
  one outstanding call per direction, timeouts, a bounded queue and feature
  registries.
- The `ws` transport: server and client, subprotocol negotiation, reconnect with
  the OCPP-J back-off, ping and pong, and security profiles 1 to 3 (Basic auth,
  TLS, client certificates).
- OCPP 1.6: all 28 messages of the Core, Firmware Management, Local Auth List
  Management, Reservation, Smart Charging and Remote Trigger profiles, and the 11
  messages of the security extension. Checked against the official JSON schemas.
- `csms/v16` and `chargepoint/v16`: the high-level APIs.
- Runnable examples, the README and the documentation site.
