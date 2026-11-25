# AGENTS.md — go-ocpp-sdk

Instructions for AI coding agents (Claude Code, Codex, etc.) working in this repo.
Read this whole file at the start of every session. It is the source of truth for
scope, layout, conventions and progress.

## 1. Project

A Go SDK for **OCPP** (Open Charge Point Protocol), the protocol between EV charging
stations and their central management system.

- Module path: `github.com/foecum/go-ocpp-sdk`
- Root package name: `ocpp`
- Go version: latest stable (set in `go.mod`)
- Licence: see `LICENSE` (ask the owner if missing)
- Roles supported: **Charge Point / Charging Station** (client) and
  **Central System / CSMS** (server)
- Protocol versions, in order: **OCPP 1.6J** → **OCPP 2.0.1** → OCPP 2.1 (later, optional)

## 2. Source of truth for the spec

- Official specs and JSON schemas go in `docs/spec/` (download from
  https://openchargealliance.org/my-oca/ocpp/; the owner adds them). JSON (OCPP-J)
  only — this SDK does not implement SOAP (OCPP-S).
  - `docs/spec/1.6/` — OCPP 1.6 edition 2 PDF, OCPP-J 1.6 specification,
    both errata sheets, `schemas/` (JSON schemas)
  - `docs/spec/2.0.1/` — OCPP 2.0.1 edition 4: parts 0–2, part 2 appendices,
    part 4 (OCPP-J), errata, `appendices-csv/`, `schemas/`; parts 5–6
    (certification profiles, test cases) for checking behaviour only
  - `docs/spec/2.1/` — OCPP 2.1 edition 2, same structure; Phase 10 only
- Order of precedence when documents disagree: errata sheet → OCPP-J spec
  (for framing/transport) → main specification → JSON schema. If a schema and
  the spec text still conflict after errata, record it in "Known gaps" and ask.
- Field names, required/optional, max string lengths, enum values and cardinality
  come from those files. **Never guess.** If the spec is missing or unclear, stop
  and ask the owner.
- The appendices CSVs (2.0.1/2.1) are the machine-readable source for standardized
  components and variables (used by GetVariables/SetVariables); prefer them over
  copying tables out of the PDF.
- Watch the spelling differences between versions in CALLERROR codes:
  1.6 uses `FormationViolation` and `OccurenceConstraintViolation` (sic);
  2.0.1 uses `FormatViolation` and `OccurrenceConstraintViolation`.

## 3. Layout

No `src/` folder. Packages live at the repo root. Never create a folder named `v2/`
(Go reserves `/v2` for module major versions).

```
go-ocpp-sdk/
├── go.mod
├── doc.go, ocpp.go         // package ocpp: shared interfaces, errors, version constants
├── ocppj/                  // OCPP-J framing (version-agnostic)
├── ws/                     // WebSocket transport (version-agnostic)
├── v16/
│   ├── types/              // shared 1.6 types + enums
│   ├── core/
│   ├── firmware/
│   ├── localauth/
│   ├── remotetrigger/
│   ├── reservation/
│   ├── smartcharging/
│   └── security/           // 1.6 security whitepaper extension (Phase 7)
├── v201/
│   ├── types/
│   └── <one package per functional block, see Phase 8>
├── chargepoint/            // high-level client API
│   ├── v16/
│   └── v201/
├── csms/                   // high-level server API
│   ├── v16/
│   └── v201/
├── internal/               // helpers not part of the public API
├── site/                   // HTML documentation site (GitHub Pages), see section 10
├── examples/
│   ├── v16/chargepoint/  v16/csms/
│   └── v201/chargingstation/  v201/csms/
└── docs/spec/
```

Message files: one file per message pair, named in snake_case after the action,
e.g. `v16/core/boot_notification.go` holds `BootNotificationRequest`,
`BootNotificationConfirmation` (1.6) — in 2.0.1 the response type is
`BootNotificationResponse`, matching each spec's naming.

## 4. Conventions

- Every function that does I/O takes `context.Context` as its first argument.
- Library code never panics on bad input; return errors and wrap with `%w`.
- Exported error values/types for the cases callers need to branch on
  (timeout, not connected, CALLERROR received, validation failed).
- Logging through `log/slog`; the caller can inject a `*slog.Logger`. Default: discard.
- Concurrency: handlers and clients must be safe for concurrent use. All tests run
  with `-race`.
- JSON tags match the spec exactly, including case. Optional fields use pointers or
  `omitempty` so absent fields are not sent.
- Enums are typed strings with constants and a `IsValid()` method.
- Date-time fields use a custom type that marshals RFC 3339 and accepts the
  variants real chargers send (with/without fractional seconds, `Z` or offset).
- Validation of required fields, string lengths and enum values happens on both
  send and receive. A malformed incoming CALL gets the correct CALLERROR, not a crash.
- Dependencies: keep them few. Standard library first. Any new dependency must be
  justified in the "Decisions" section below before it is added.
- Every exported identifier has a doc comment. Each package has a `doc.go` with a
  short usage example.
- Formatting and linting: `gofmt`, `go vet`, and `staticcheck` (if installed) are clean.
- **Test coverage (mandatory):** unit test statement coverage MUST NEVER fall below
  **95%**, measured per package (not just in total), and every critical path MUST be
  covered. Critical paths are: frame parsing and serialization, every error and
  CALLERROR path, validation of required fields/lengths/enums on send and receive,
  timeouts, cancellation, disconnects, reconnects, queueing and ordering,
  authentication and TLS handling, and concurrent use. A change that drops any
  package below 95% is not done; add tests, never lower the threshold or exclude
  code to pass it. Only `examples/`, the test-support package `internal/schematest`
  and packages with no statements (doc-only) are exempt. `scripts/coverage.sh` enforces this and CI runs it.

## 5. Commands

Run after every change, and fix failures before moving on:

```bash
go build ./...
go vet ./...
go test -race ./...
scripts/coverage.sh        # fails if any package is below 95% coverage
```

The OCPP 2.0.1 sources under `v201/` are generated from the JSON schemas. After changing
the generator (`internal/gen201`) or the schemas, regenerate them (the test fails when the
committed files are out of date):

```bash
go test ./internal/gen201 -run TestGeneratedCodeIsUpToDate -update
```

Optional when available: `staticcheck ./...`, `golangci-lint run`.

## 6. How to work

1. At the start of a session, read this file and find the first unchecked item in
   the Roadmap. Work on that item only.
2. Before starting a new **phase**, post a short plan (files, public API sketch,
   open questions) and wait for the owner's OK.
3. Work in small steps. After each step: run the commands in section 5, then commit.
4. Commit messages: `<area>: <what changed>` e.g. `ocppj: parse CALLERROR frames`.
   One logical change per commit.
5. When an item is done, tick its checkbox in this file in the same commit.
6. Record any design decision or deviation from the spec in "Decisions" below.
7. Stop and ask the owner when: the spec is unclear, a new dependency is needed,
   the public API of `chargepoint`/`csms` needs to change after it was agreed, or
   a test cannot be made to pass without changing agreed behaviour.
8. Keep the documentation site in sync (section 10): any change that adds, removes
   or alters public API, behaviour, supported messages, configuration, errors or
   roadmap status MUST update `site/` in the same commit.
9. Never delete or weaken a test to make it pass, and never lower the 95% coverage
   threshold (section 4).

## 7. Roadmap

### Phase 0 — Setup
- [x] `go.mod` with module path `github.com/foecum/go-ocpp-sdk`
- [x] `README.md` stub, `LICENSE` (ask owner which), `.gitignore`
- [x] CI: GitHub Actions running build, vet, `test -race`, staticcheck on push and PR
- [x] `docs/spec/README.md` listing which spec files are expected there

### Phase 1 — Core (`ocpp`, `ocppj`)
- [x] `ocpp` package: `Request`, `Response`, `Feature` interfaces; version constants;
      shared error types
- [x] `ocppj`: parse/serialize CALL `[2, id, action, payload]`,
      CALLRESULT `[3, id, payload]`, CALLERROR `[4, id, code, description, details]`
- [x] Unique message IDs (max 36 chars)
- [x] Pending-request tracker: correlate responses by ID, per-request timeout,
      one outstanding CALL per direction per connection (spec rule), queue the rest
- [x] Feature registry: map action name → request/response types, per version
- [x] CALLERROR code sets for 1.6 and 2.0.1 (note spelling differences)
- [x] Table-driven tests: valid frames, malformed JSON, wrong element types,
      unknown action, unknown message type, response with unknown ID, timeout

### Phase 2 — Transport (`ws`)
- [x] Pick the WebSocket library (`github.com/coder/websocket` vs
      `github.com/gorilla/websocket`) and record why in Decisions
- [x] Server: charge point ID from the last URL path segment; connection
      registry; connect/disconnect callbacks
- [x] Subprotocol negotiation (`ocpp1.6`, `ocpp2.0.1`); reject if none match
- [x] Client: connect, auto-reconnect with exponential backoff + jitter
- [x] Ping/pong keep-alive with configurable interval
- [x] Security profile 1: HTTP Basic Auth (charge point ID as username)
- [x] Security profile 2: TLS + Basic Auth
- [x] Security profile 3: TLS with client certificates (mTLS)
- [x] Graceful shutdown for server and client
- [x] Tests: handshake, subprotocol mismatch, auth failure, reconnect,
      concurrent connections

### Phase 3 — OCPP 1.6 types and Core profile
- [x] `v16/types`: IdTagInfo, AuthorizationStatus, MeterValue, SampledValue
      (+ Measurand, Phase, Location, Unit, ReadingContext, ValueFormat enums),
      ChargingProfile, ChargingSchedule, ChargingSchedulePeriod, KeyValue,
      ChargePointStatus, ChargePointErrorCode, DateTime
- [x] `v16/core`: Authorize, BootNotification, ChangeAvailability,
      ChangeConfiguration, ClearCache, DataTransfer, GetConfiguration, Heartbeat,
      MeterValues, RemoteStartTransaction, RemoteStopTransaction, Reset,
      StartTransaction, StatusNotification, StopTransaction, UnlockConnector
- [x] JSON round-trip test for every message using sample payloads
- [x] Validate generated JSON against the official 1.6 JSON schemas in tests

### Phase 4 — High-level API for 1.6
- [x] Post the API sketch for `chargepoint/v16` and `csms/v16` and wait for OK
- [x] `chargepoint/v16`: typed methods for CP-initiated calls
      (e.g. `BootNotification(ctx, req) (*core.BootNotificationConfirmation, error)`)
      and handler interfaces for CSMS-initiated calls
- [x] `csms/v16`: typed handlers for CP-initiated calls; typed methods to send
      calls to a charge point by ID; list connected charge points
- [x] Unhandled action → CALLERROR `NotImplemented`; registered profile with no
      handler → `NotSupported`
- [x] Integration test (in-process): BootNotification → StatusNotification →
      Authorize → StartTransaction → MeterValues → StopTransaction,
      plus RemoteStartTransaction and Reset from the CSMS
- [x] Tests for timeout, disconnect mid-call, reconnect and resend

### Phase 5 — Remaining 1.6 profiles
- [x] Firmware Management: GetDiagnostics, DiagnosticsStatusNotification,
      FirmwareStatusNotification, UpdateFirmware
- [x] Local Auth List Management: GetLocalListVersion, SendLocalList
- [x] Reservation: ReserveNow, CancelReservation
- [x] Smart Charging: SetChargingProfile, ClearChargingProfile, GetCompositeSchedule
- [x] Remote Trigger: TriggerMessage
- [x] Wire all profiles into `chargepoint/v16` and `csms/v16`; tests for each

### Phase 6 — 1.6 examples and docs
- [x] `examples/v16/csms` and `examples/v16/chargepoint` that talk to each other
- [x] README: install, quick start for both roles, supported messages table,
      security profiles, logging, error handling
- [x] Package docs with examples (`Example...` test functions)
- [x] Tag `v0.1.0` (ask owner first)

### Phase 7 — OCPP 1.6 security extension (optional, ask owner)
- [x] `v16/security`: SecurityEventNotification, SignCertificate,
      CertificateSigned, InstallCertificate, DeleteCertificate,
      GetInstalledCertificateIds, ExtendedTriggerMessage, GetLog, LogStatusNotification,
      SignedUpdateFirmware, SignedFirmwareStatusNotification

### Phase 8 — OCPP 2.0.1
- [x] Post a plan for 2.0.1 package split and wait for OK
- [x] `v201/types`: shared data types and enums (generate from JSON schemas if
      practical — record the generator approach in Decisions)
- [x] Functional blocks, one package each:
  - [x] A Security
  - [x] B Provisioning
  - [x] C Authorization
  - [x] D Local Authorization List
  - [x] E Transactions
  - [x] F Remote Control
  - [x] G Availability
  - [x] H Reservation
  - [x] I Tariff and Cost
  - [x] J Meter Values
  - [x] K Smart Charging
  - [x] L Firmware Management
  - [x] M ISO 15118 Certificate Management
  - [x] N Diagnostics
  - [x] O Display Message
  - [x] P Data Transfer
- [x] `chargepoint/v201` (charging station) and `csms/v201` high-level APIs
- [x] Validate every message against the official 2.0.1 JSON schemas in tests
- [x] Integration test: boot, authorize, TransactionEvent (Started/Updated/Ended),
      RequestStartTransaction from CSMS, GetVariables/SetVariables
- [x] `examples/v201/*` and README update
- [x] Tag `v0.2.0` (ask owner first)

### Phase 9 — Hardening
- [x] Fuzz tests for `ocppj` parsing (`go test -fuzz`)
- [x] Benchmarks for framing and a CSMS with many simulated charge points
- [x] Load test example: 1,000 simulated charge points against one CSMS
- [ ] Review public API for consistency; write `CHANGELOG.md`  (review: docs/api-review.md, awaiting decisions)
- [ ] Tag `v1.0.0` (ask owner first)

### Phase 10 — OCPP 2.1 (later, ask owner)
- [ ] Plan only when the owner asks

## 8. Decisions

Record each decision as: date — decision — reason.

- 2026-11-25 — Module path `github.com/foecum/go-ocpp-sdk`, root package `ocpp`.
- 2026-11-25 — Licence: MIT — chosen by the owner.
- 2026-11-25 — `go` directive follows the installed toolchain (1.27.2).
- 2026-11-25 — CI runs staticcheck via `go run honnef.co/go/tools/cmd/staticcheck@latest`, so no module dependency is added.
- 2026-11-25 — Coverage floor of 95% per package plus full critical-path coverage, enforced by `scripts/coverage.sh` and CI — set by the owner.
- 2026-11-25 — `ErrorCode` and the per-version code sets live in the root `ocpp` package (not `ocppj`) so `ocpp.CallError` can use them without an import cycle. OCPP 2.1 uses the 2.0.1 code set (checked against 2.1 OCPP-J table 8).
- 2026-11-25 — Validation is hand-written `Validate()` methods on requests/responses, no validator dependency.
- 2026-11-25 — Framing rules: unreadable message ID is answered with `-1` (2.0.1 OCPP-J 4.2.3); unknown message type gets `MessageTypeNotSupported` in 2.0.1/2.1 and is ignored in 1.6; a `null` payload is accepted and treated as `{}`.
- 2026-11-25 — `ocppj.Endpoint` is transport-agnostic (the transport passes a `Send` func and feeds `HandleFrame`). Defaults: 30s response timeout counted from send, 1000-call bounded queue. A caller whose context ends frees the outstanding slot (treated like a timeout) so the queue keeps moving; late responses are dropped. Incoming CALLs run in their own goroutine so handlers may call back into the peer.
- 2026-11-25 — Documentation is a self-contained static site in `site/`, deployed to GitHub Pages by `.github/workflows/pages.yml` (only `site/` is published, never `docs/spec/`). Updating it is part of every change that affects users (section 10) — set by the owner.
- 2026-11-25 — WebSocket library: `github.com/coder/websocket` (v1.8.15). Reasons: actively maintained (gorilla's last release was June 2024), context-aware reads/writes/pings/dials (our convention: `context.Context` first), safe concurrent writes (gorilla needs a manual write mutex), standard-library-only dependencies, built-in `Subprotocols` negotiation. Approved by the owner.
- 2026-11-25 — Transport decisions: `ws.Server` is an `http.Handler`; version is chosen by the server's preference order among the client's offered subprotocols; a subprotocol mismatch completes the handshake without a subprotocol and then closes with 1002 (OCPP-J §3.2, same text in 1.6 and 2.0.1); a duplicate connection for the same ID replaces the old one (closed with 1008); ID rules: ≤48 chars, no `:` or `/`; unknown ID → 404, bad credentials → 401, failed certificate identity → 403; TLS ≥1.2 with the spec's GCM suites (plus ECDHE-RSA GCM); profile 3 requires subject CN == charge point ID and optionally O containing the operator name (A00.FR.404/405/511); WebSocket compression disabled; binary frames rejected; defaults: ping 30s, pong timeout 10s, read limit 1 MiB.
- 2026-11-25 — `ws.Client` reconnects per OCPP-J §5.4 using the standard variable names (`WaitMinimum`, `RandomRange`, `RepeatTimes`); defaults 1s / 2s / 6 doublings when the config is the zero value. The wait is applied after every lost connection as well as after failed attempts; a missing common subprotocol is treated as a failed attempt and retried.
- 2026-11-25 — 1.6 message types: Go-style field names (`IDTag`, `ConnectorID`, `TransactionID`, `IDTagInfo`) with the exact spec JSON tags; requests are `<Action>Request`, responses `<Action>Confirmation` (spec ".req"/".conf" naming). Optional strings/enums use the empty string for "absent" (`omitempty`); optional numbers and timestamps are pointers because zero is meaningful; required timestamps are a non-pointer `types.DateTime` whose zero value counts as missing.
- 2026-11-25 — Required fields are marked `ocpp:"required"` and checked against the raw JSON by `ocppj.Registry` (at any depth), because Go cannot tell a missing int/enum from its zero value. `Validate()` then checks lengths, ranges, enums and cardinalities (e.g. `connectorId` ≥ 0 or > 0 as the spec says per message, `meterValue` 1..*, first charging period starts at 0, limits with at most one decimal, RemoteStartTransaction profile purpose must be TxProfile). These spec rules are stricter than the JSON schemas where the spec says so (precedence: errata → OCPP-J → spec → schema).
- 2026-11-25 — Decoding is deliberately lenient about unknown properties (the schemas say `additionalProperties: false`): real charge points send extras and rejecting them would break interoperability. Everything else the schemas reject is rejected too; `internal/schematest` proves it with mutation tests against the official schemas.
- 2026-11-25 — `types.DateTime` marshals UTC (whole seconds, or milliseconds when present) and accepts RFC 3339 with/without fractions, `Z` or offsets, and no zone (taken as UTC) — set by the owner. `Celcius` and `Celsius` are both valid `UnitOfMeasure` values (OCPP-J 1.6 errata 2025-04); the SDK sends whatever the caller sets.
- 2026-11-25 — New test-only dependency: `github.com/santhosh-tekuri/jsonschema/v6` v6.0.3 (draft-04 support; pulls `golang.org/x/text`). Used only by `internal/schematest`; approved by the owner.
- 2026-11-25 — High-level API (approved by the owner): packages `csms/v16` and `chargepoint/v16` are named `csms` and `chargepoint`; handlers are one interface per profile (`CoreHandler`) with embeddable `Unimplemented...` structs that answer `NotSupported`; a nil profile handler also answers `NotSupported`, an action the SDK does not know answers `NotImplemented`. CSMS handlers receive the `*csms.ChargePoint` so they can call the station back; outbound requests are passed by value, inbound ones by pointer. A `*csms.ChargePoint` is tied to one connection. A panicking handler is answered with `InternalError`; handler error text is never sent to the peer (only `*ocpp.CallError` is passed through).
- 2026-11-25 — Offline behaviour (approved by the owner): calls fail fast with `ocpp.ErrNotConnected` (also for a call interrupted by a lost connection) and nothing is queued or retried by the SDK; the station never sends BootNotification or Heartbeat on its own. An opt-in retry helper for transaction messages (1.6 `TransactionMessageAttempts` / `TransactionMessageRetryInterval`) is planned for later.
- 2026-11-25 — Shared helpers for the two packages live in `internal/rpc`. Construction seams (`newRegistry`, `newEndpoint`, `newWSServer`) are package variables that tests replace to reach error paths that cannot occur otherwise.
- 2026-11-25 — Remaining 1.6 profiles: one package per profile (`firmware`, `localauth`, `reservation`, `smartcharging`, `remotetrigger`), each with `Features()`; `v16.NewRegistry()` registers all 28 messages. `GetDiagnostics`/`UpdateFirmware` locations must be absolute URIs (the schemas' `uri` format). Rules from the spec text enforced beyond the schemas: `SendLocalList` idTags unique (case-insensitive) and `idTagInfo` present on every entry of a Full update; `GetLocalListVersion` returns ≥ -1; `TriggerMessage.connectorId` > 0 when present; connector ids ≥ 0 for ReserveNow, SetChargingProfile, ClearChargingProfile and GetCompositeSchedule. `SendLocalList.listVersion` is not constrained (the spec states no limit).
- 2026-11-25 — Handler wiring for those profiles follows the approved Phase 4 shape: one optional `Config` field per profile (`csms.Config.Firmware`; `chargepoint.Config.Firmware/LocalAuth/Reservation/SmartCharging/RemoteTrigger`), a nil one answers `NotSupported`. Profiles whose messages only flow from the Central System need no CSMS-side handler. Shared test helpers live in `internal/schematest`; profile `Features()` are built with `internal/featureset`.
- 2026-11-25 — Examples (approved by the owner): plain `ws://` with optional Basic auth, no TLS flags (the README shows the TLS setup). Each example is a thin `main` over `run(ctx, ...)`; `examples/v16/smoke_test.go` builds both and runs them against each other (skipped with `-short`). Every package has tested `Example` functions with `// Output:` checks. The `v0.1.0` tag is made only with the owner's OK, preferably after pushing and seeing CI and Pages succeed.
- 2026-11-25 — `v0.1.0` tagged locally with the owner's OK (OCPP 1.6J complete). Not pushed: the owner pushes `main` and the tag, after CI and Pages have run.
- 2026-11-25 — Security extension (approved by the owner): all its types live in `v16/security` (not in the shared `types`); its enumerations are generated from the whitepaper schemas and renamed where they clash with Core ones (`SignedFirmwareStatus`, `ExtendedMessageTrigger`, `SignedUpdateFirmwareStatus`); security event names are untyped string constants with `IsKnownEvent` / `IsCriticalEvent`; wiring is a `Security` field in both `Config`s with `Unimplemented...` handlers (Central System side: security events, SignCertificate, log and firmware status; station side: certificates, GetLog, SignedUpdateFirmware, ExtendedTriggerMessage). `internal/schematest` inlines `$ref` so the draft-06 schemas can be tested.
- 2026-11-25 — No Claude/AI attribution trailers in commits (owner's request): the local history was rewritten to remove `Co-Authored-By` lines. Commits up to the first push (`1f8f63f`, tag `v0.1.0`) were already on the remote, so publishing the rewrite needs a force push by the owner.
- 2026-11-25 — OCPP 2.0.1 types and messages are generated (approved by the owner): `internal/gen201` reads `docs/spec/2.0.1/schemas` and writes `v201/types` (143 shared definitions: structs with `Validate()`, enums with `IsValid()`) and one package per functional block (`v201/<block>`: `XRequest`/`XResponse`, `Validate()`, `Features()`, `doc.go`), plus `v201/v201.go` (`NewRegistry`). Generated files carry the standard "Code generated ... DO NOT EDIT" header and are committed; `TestGeneratedCodeIsUpToDate` fails if they drift (run with `-update` to rewrite). The block table and request directions are in `internal/gen201/tables.go` (64 messages: 24 from the station, 39 from the CSMS, DataTransfer both).
- 2026-11-25 — 2.0.1 conventions: Go names strip the schema's `Type`/`EnumType` suffix and use initialisms (`IDToken`, `EVSEID`, `OCPPCSMSURL`); an enum whose name an object already has keeps `Enum` (`IDTokenEnum`); a field whose name clashes with the `Action` method is renamed (`CertificateAction`). Optional scalars/structs/timestamps are pointers, optional strings/enums use the empty value, arrays are always `omitempty` (an absent optional array is fine, a present one must meet minItems/maxItems). A required string must be non-empty except the allow-listed `SetVariableData.attributeValue`. Properties without a type (DataTransfer `data`) are `json.RawMessage`. `customData` is `*types.CustomData`, which keeps unknown vendor properties. Decimals are `float64`.
- 2026-11-25 — `ocpp.DateTime` (root package) is the one DateTime type for every version; `v16/types.DateTime` and `v201/types.DateTime` are aliases of it. 2.0.1 validation helpers are in `internal/check`.
- 2026-11-25 — Schema tests for 2.0.1 derive minimal and full sample payloads from each schema (`schematest.RunGenerated`) and break them in every way the schema forbids (required, maxLength, enum, type, date-time, minItems/maxItems, minimum/maximum). `schematest.AllowMutation` lists the rare cases where the spec text overrides the schema (1.6 `GetInstalledCertificateIds.conf` may be empty).
- 2026-11-25 — Bug fix after v0.1.0: a connection whose handshake completed after `ws.Server.Shutdown` had taken its snapshot of open connections was never closed, so Shutdown waited for its whole context (seen as a flaky `TestServeAndShutdown` in CI). Such a connection is now closed with 1001 as it registers. The tagged v0.1.0 contains the bug; a v0.1.1 is worth tagging (ask the owner).
- 2026-11-25 — Shared lifecycle (no behaviour change for 1.6): the connection handling common to `csms/v16` and `csms/v201` lives in `internal/csmscore` (generic `Server[H]`, `Session`, `Call`), and the one common to `chargepoint/v16` and `chargepoint/v201` in `internal/cpcore` (`Station`, `Call`). The public `ChargePoint` types hold them in an unexported field and forward `ID`, `Version`, `RemoteAddr`, `Done`, `Close` (CSMS side) and `Run`, `Close`, `Connected`, `WaitConnected` (station side) explicitly, so the public method sets are exactly those of v0.1.0 and `internal` types do not leak into the documentation. (They were embedded for a few commits; the API review caught that.)
- 2026-11-25 — 2.0.1 high-level APIs: the handler interfaces, `Unimplemented...` structs, dispatch and typed calls are generated by `internal/gen201` (`highlevel.go`) from the block table and the direction table, so they cannot drift from the messages; `Config`, `CSMS` and `ChargePoint` are hand-written. One handler interface per functional block that has messages the side receives (`<Name>Handler`, `Config.<Name>`; names in `gen201.HandlerNames`); blocks with nothing to receive on a side (CSMS: local auth list, remote control, tariff and cost; station: meter values) have no field there. Same rules as 1.6: nil or unimplemented handler answers `NotSupported`, unknown action `NotImplemented`. Tests derive their expectations from `gen201.Directions` by reflection, and `csms/v201` has the end-to-end scenario test.
- 2026-11-25 — 2.0.1 examples (`examples/v201/csms`, `examples/v201/chargingstation`) mirror the 1.6 ones: plain `ws://` with optional Basic auth, in-memory state only (the station keeps a tiny variable map, not a device model), only the public API; `examples/v201/smoke_test.go` builds both and runs a remote-started session end to end (skipped with `-short`). Neither example has `Example` functions; the smoke test is their test.
- 2026-11-25 — `v0.2.0` tagged locally with the owner's OK (OCPP 2.0.1 messages, high-level APIs and examples). Not pushed: the owner pushes `main` and the tag, after CI and Pages have run. The tag also carries the `ws.Server` shutdown fix, so it supersedes the planned v0.1.1.
- 2026-11-25 — Fuzzing: six targets (`ocppj` FuzzParse / FuzzRegistryDecode / FuzzEndpointHandleFrame, `ocpp` FuzzDateTime, `v16` and `v201` FuzzDecode over every message) check for panics and for accept-then-roundtrip consistency. Their seeds run as ordinary tests; `.github/workflows/fuzz.yml` fuzzes each for 60 s weekly and on demand (approved by the owner). A failing input is uploaded as an artifact and must be committed under `testdata/fuzz` with its fix.
- 2026-11-25 — Benchmarks (`ocppj`, `v16`, `v201`, `csms/v16`, `csms/v201`) have no CI thresholds; indicative results are kept in `docs/benchmarks.md`, labelled with the machine (approved by the owner).
- 2026-11-25 — Load test (`examples/loadtest`): in-process CSMS plus N simulated stations over loopback WebSockets, for 1.6 and 2.0.1; reports operations/s and latency percentiles. A 20-station smoke test runs in the suite (skipped with `-short`); the 1,000-station run is manual.
- 2026-11-25 — Version packages named `v16`, `v201`, `v21`; never `v2/`.

## 9. Known gaps / skipped spec items

List anything not implemented or simplified, with a reason.

- (none yet)
- Opt-in retry/queue helper for transaction messages (StartTransaction, StopTransaction, MeterValues) per the 1.6 configuration keys: not implemented; the owner wants it added later (it needs persistence decisions).
- 2.0.1: constraints stated only in the Part 2 field tables (not in the schemas) are not yet enforced; the review of those tables is still to do. Standardized components/variables, reason codes and security events (appendices CSVs) are not yet available as constants.
- 1.6 string types: the specification restricts `CiString` and `String` to printable ASCII. Only lengths are enforced (schemas do the same), because real charge points send other characters. Revisit if strictness is wanted.
- 1.6 security profiles: resolved by the security whitepaper (edition 3, now in `docs/spec/1.6/`). The Basic-auth password is the `AuthorizationKey` (16–20 bytes as 32–40 hex digits, sent as that text), which `ws.BasicAuthenticator` already compares as given; `security.CheckAuthorizationKey` / `GenerateAuthorizationKey` validate and create keys. Still open: the whitepaper tolerates TLS 1.0/1.1 for old stations; `ws` requires TLS 1.2 or newer and does not support them.
- 1.6 security extension, leniencies: `serialNumber` in `CertificateHashData` is checked for hexadecimal characters only (the whitepaper says no leading zeros; stations that send them are accepted); `GetInstalledCertificateIds.conf` accepts an empty `certificateHashData` list (the spec says 0..*, the schema says at least 1; the spec wins); certificates and CSRs are checked as PEM blocks of the right type but their X.509 contents are not parsed.

## 10. Documentation site

User-facing documentation is a static HTML site in `site/` (entry point
`site/index.html`), published to GitHub Pages by `.github/workflows/pages.yml`.
Only `site/` is published, never `docs/spec/` (the OCA specifications are not ours
to redistribute).

- **Update `site/` every time a change adds something users need to know**: new or
  changed public API, new packages, supported messages (keep the supported-messages
  table accurate), configuration options, error values and how to handle them,
  security profiles, behaviour changes, and roadmap/status changes. Do it in the
  same commit as the code, and update README.md accordingly.
- Every code sample on the site must compile and run against the current API. Check
  samples before committing (paste into a scratch `main.go`, run, delete it).
- Document only what exists. Anything planned is labelled "planned" and must not
  look usable. Bump the status table when a roadmap phase completes.
- Keep it self-contained: one HTML file plus assets in `site/`, no build step, no
  external scripts; system fonts only. It must work in light and dark mode and on
  phone widths.
- Before merging a docs-only or docs-affecting change, open `site/index.html` in a
  browser and check it renders correctly.
