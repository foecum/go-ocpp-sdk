# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The project is
stable from 1.0.0 on and follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.1.0] - 2026-11-26

### Added
- OCPP 2.1: all 91 messages in 19 functional blocks, including the new blocks
  Bidirectional Power Transfer (`v21/bidirectional`), DER Control (`v21/dercontrol`)
  and Battery Swapping (`v21/batteryswap`), generated from the official JSON schemas
  (`v21`, `v21/types`, `v21/<block>`), with `csms/v21`, `chargepoint/v21`,
  `examples/v21/*` and `examples/loadtest -version 2.1`.
- OCPP-J message types of 2.1: SEND, an unconfirmed message without a response
  (`ocppj.Send`, `Endpoint.Send`, `Config.SendHandler`), and CALLRESULTERROR
  (`ocppj.CallResultErrorFrame`, `Config.ResultErrorHandler`). A CALLRESULT that cannot
  be processed is answered with a CALLRESULTERROR on 2.1 connections. Incoming SEND
  messages reach the handler one at a time, in order. `ocpp.NewSendFeature`,
  `ocpp.OneWayFeature` and `ocpp.IsOneWay` describe messages without a response.
- `NotifyPeriodicEventStream` (2.1) is the one SEND message: `ChargePoint.NotifyPeriodicEventStream`
  sends it and `csms` handlers receive it with `NotifyPeriodicEventStream(ctx, cp, msg) error`.
- `v201/appendices` and `v21/appendices`: the standardized components, variables,
  device model, units of measure, reason codes and security events of the
  specification appendices (and for 2.1 the idToken, connector, payment and signing
  method lists) as constants and tables, generated from the specification's CSV files.
- `v201` and `v21` validate constraints the specification text states but the JSON
  schemas lack: for example `SendLocalList` entries carry `idTokenInfo` on a Full
  update, `evseId` greater than 0 where required, `valuesList` for list-typed
  variables, a `location` for a Published firmware, one decimal in a charging limit,
  `phaseToUse` 1..3, `severity` 0..9; for 2.1 also the V2X curves, `randomizedDelay`,
  the required `pctMaxDischargePower` (errata 2026-09), the syntax of the time of
  day and dates in tariff conditions.
- `chargepoint/v16`: `TransactionQueue` delivers StartTransaction, StopTransaction and
  MeterValues in order, keeps them while the station is offline, retries what the
  Central System fails to process (`TransactionMessageAttempts`,
  `TransactionMessageRetryInterval`) and can keep the queue in a file (`FileStore`).
- `CSMS.ForEach` in `csms/v16`, `csms/v201` and `csms/v21` visits every connected station.

### Changed
- On 2.1 connections an unknown message type is ignored, and `MessageTypeNotSupported`
  is no longer sent (OCPP 2.1 errata 2026-09).
- The generator is now `internal/gen` and produces both `v201` and `v21`.

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
  `internal/gen201` (now `internal/gen`) and tested against them.
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
