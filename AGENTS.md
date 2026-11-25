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
  code to pass it. Only `examples/` and packages with no statements (doc-only) are
  exempt. `scripts/coverage.sh` enforces this and CI runs it.

## 5. Commands

Run after every change, and fix failures before moving on:

```bash
go build ./...
go vet ./...
go test -race ./...
scripts/coverage.sh        # fails if any package is below 95% coverage
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
- [ ] Pick the WebSocket library (`github.com/coder/websocket` vs
      `github.com/gorilla/websocket`) and record why in Decisions
- [ ] Server: charge point ID from the last URL path segment; connection
      registry; connect/disconnect callbacks
- [ ] Subprotocol negotiation (`ocpp1.6`, `ocpp2.0.1`); reject if none match
- [ ] Client: connect, auto-reconnect with exponential backoff + jitter
- [ ] Ping/pong keep-alive with configurable interval
- [ ] Security profile 1: HTTP Basic Auth (charge point ID as username)
- [ ] Security profile 2: TLS + Basic Auth
- [ ] Security profile 3: TLS with client certificates (mTLS)
- [ ] Graceful shutdown for server and client
- [ ] Tests: handshake, subprotocol mismatch, auth failure, reconnect,
      concurrent connections

### Phase 3 — OCPP 1.6 types and Core profile
- [ ] `v16/types`: IdTagInfo, AuthorizationStatus, MeterValue, SampledValue
      (+ Measurand, Phase, Location, Unit, ReadingContext, ValueFormat enums),
      ChargingProfile, ChargingSchedule, ChargingSchedulePeriod, KeyValue,
      ChargePointStatus, ChargePointErrorCode, DateTime
- [ ] `v16/core`: Authorize, BootNotification, ChangeAvailability,
      ChangeConfiguration, ClearCache, DataTransfer, GetConfiguration, Heartbeat,
      MeterValues, RemoteStartTransaction, RemoteStopTransaction, Reset,
      StartTransaction, StatusNotification, StopTransaction, UnlockConnector
- [ ] JSON round-trip test for every message using sample payloads
- [ ] Validate generated JSON against the official 1.6 JSON schemas in tests

### Phase 4 — High-level API for 1.6
- [ ] Post the API sketch for `chargepoint/v16` and `csms/v16` and wait for OK
- [ ] `chargepoint/v16`: typed methods for CP-initiated calls
      (e.g. `BootNotification(ctx, req) (*core.BootNotificationConfirmation, error)`)
      and handler interfaces for CSMS-initiated calls
- [ ] `csms/v16`: typed handlers for CP-initiated calls; typed methods to send
      calls to a charge point by ID; list connected charge points
- [ ] Unhandled action → CALLERROR `NotImplemented`; registered profile with no
      handler → `NotSupported`
- [ ] Integration test (in-process): BootNotification → StatusNotification →
      Authorize → StartTransaction → MeterValues → StopTransaction,
      plus RemoteStartTransaction and Reset from the CSMS
- [ ] Tests for timeout, disconnect mid-call, reconnect and resend

### Phase 5 — Remaining 1.6 profiles
- [ ] Firmware Management: GetDiagnostics, DiagnosticsStatusNotification,
      FirmwareStatusNotification, UpdateFirmware
- [ ] Local Auth List Management: GetLocalListVersion, SendLocalList
- [ ] Reservation: ReserveNow, CancelReservation
- [ ] Smart Charging: SetChargingProfile, ClearChargingProfile, GetCompositeSchedule
- [ ] Remote Trigger: TriggerMessage
- [ ] Wire all profiles into `chargepoint/v16` and `csms/v16`; tests for each

### Phase 6 — 1.6 examples and docs
- [ ] `examples/v16/csms` and `examples/v16/chargepoint` that talk to each other
- [ ] README: install, quick start for both roles, supported messages table,
      security profiles, logging, error handling
- [ ] Package docs with examples (`Example...` test functions)
- [ ] Tag `v0.1.0` (ask owner first)

### Phase 7 — OCPP 1.6 security extension (optional, ask owner)
- [ ] `v16/security`: SecurityEventNotification, SignCertificate,
      CertificateSigned, InstallCertificate, DeleteCertificate,
      GetInstalledCertificateIds, ExtendedTriggerMessage, GetLog, LogStatusNotification,
      SignedUpdateFirmware, SignedFirmwareStatusNotification

### Phase 8 — OCPP 2.0.1
- [ ] Post a plan for 2.0.1 package split and wait for OK
- [ ] `v201/types`: shared data types and enums (generate from JSON schemas if
      practical — record the generator approach in Decisions)
- [ ] Functional blocks, one package each:
  - [ ] A Security
  - [ ] B Provisioning
  - [ ] C Authorization
  - [ ] D Local Authorization List
  - [ ] E Transactions
  - [ ] F Remote Control
  - [ ] G Availability
  - [ ] H Reservation
  - [ ] I Tariff and Cost
  - [ ] J Meter Values
  - [ ] K Smart Charging
  - [ ] L Firmware Management
  - [ ] M ISO 15118 Certificate Management
  - [ ] N Diagnostics
  - [ ] O Display Message
  - [ ] P Data Transfer
- [ ] `chargepoint/v201` (charging station) and `csms/v201` high-level APIs
- [ ] Validate every message against the official 2.0.1 JSON schemas in tests
- [ ] Integration test: boot, authorize, TransactionEvent (Started/Updated/Ended),
      RequestStartTransaction from CSMS, GetVariables/SetVariables
- [ ] `examples/v201/*` and README update
- [ ] Tag `v0.2.0` (ask owner first)

### Phase 9 — Hardening
- [ ] Fuzz tests for `ocppj` parsing (`go test -fuzz`)
- [ ] Benchmarks for framing and a CSMS with many simulated charge points
- [ ] Load test example: 1,000 simulated charge points against one CSMS
- [ ] Review public API for consistency; write `CHANGELOG.md`
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
- 2026-11-25 — Version packages named `v16`, `v201`, `v21`; never `v2/`.

## 9. Known gaps / skipped spec items

List anything not implemented or simplified, with a reason.

- (none yet)

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
