# Public API review (before v1.0.0)

Reviewed 2026-11-25 for consistency between OCPP 1.6 and 2.0.1, naming, doc
comments and conventions in `AGENTS.md`. The packages reviewed are `ocpp`,
`ocppj`, `ws`, `csms/v16`, `csms/v201`, `chargepoint/v16`, `chargepoint/v201`,
`v16` and `v201`.

## Fixed in this review

- The public `ChargePoint` types of the four high-level packages embedded
  `*csmscore.Session` and `*cpcore.Station` from `internal`. That exposed an
  exported field of a type nobody can import, hid the methods on pkg.go.dev, and
  added `Endpoint()`. They now forward the documented methods explicitly, and the
  method sets are exactly those of v0.1.0.

## Checked and fine

- Every exported identifier in the public packages has a doc comment (checked with
  a script, generated code included).
- 1.6 and 2.0.1 high-level APIs have the same shape: `New(Config)`, one handler
  interface per profile or block, embeddable `Unimplemented...`, nil handler
  answers `NotSupported`, typed calls, `ErrNotConnected` while offline, panics
  become `InternalError`.
- Errors branch with `errors.Is` and `errors.As` on the same values for both
  versions.
- The spec's own naming is kept where it differs (`...Confirmation` in 1.6,
  `...Response` in 2.0.1; `FormationViolation` and `FormatViolation`).

## Proposed changes and decisions

Each of these changes the agreed public API. The owner approved the
recommendations on 2026-11-25: items 1 to 3 are applied, 4 to 7 are kept or
deferred as recommended.

1. **(Applied: renamed `ocppj.CallErrorFrame`.) Two types called `CallError`** (`ocpp.CallError`, the error a call returns, and
   `ocppj.CallError`, the CALLERROR frame). Easy to confuse in a signature or an
   import. Option: rename the frame type to `ocppj.CallErrorFrame`, or leave and
   document. Recommend: rename; `ocppj` is the low-level package, so few users
   are affected.
2. **(Applied: renamed `ChargePointIDs()`.) `CSMS.ChargePoints()` returned identities, not charge points**, while
   `CSMS.ChargePoint(id)` returns one. The names suggest the same kind of value.
   Option: rename to `ChargePointIDs()`. Recommend: rename.
3. **(Applied: exception recorded in `AGENTS.md`.) `Close()` and I/O without a context.** `AGENTS.md` says functions that do I/O
   take a `context.Context`. `ChargePoint.Close()` (station and CSMS side) closes a
   connection gracefully without one. `CSMS.Shutdown(ctx)` does take one.
   Option: leave (a close handshake has its own short timeout) and record the
   exception in `AGENTS.md`, or add `CloseContext(ctx)`. Recommend: record the
   exception.
4. **`Now()` in two places.** `ocpp.Now()` is the one definition; `v16/types.Now()`
   is an alias kept for convenience. Both work and return the same type. Option:
   keep. Recommend: keep, and use `ocpp.Now()` in new docs.
5. **Station called `ChargePoint` in 2.0.1.** The 2.0.1 specification says
   "Charging Station". The type is `ChargePoint` in both versions so that code
   moves between them. Option: keep, with the package comment saying so.
   Recommend: keep.
6. **No broadcast or iteration helper on the CSMS** beyond looking up by identity.
   Users who want to call every station need `ChargePointIDs()` plus lookups,
   which races with disconnects. Option: add `ForEach(func(*ChargePoint))`
   later, additively, after v1.0.0. Recommend: defer; it is additive.
7. **The opt-in retry/queue helper for transaction messages** (a known gap in
   `AGENTS.md`) would add API. Decide whether it ships in v1.0.0 or later; it is
   additive either way.

Nothing else in the public API needs to change before v1.0.0.
