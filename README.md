# go-ocpp-sdk

[![CI](https://github.com/foecum/go-ocpp-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/foecum/go-ocpp-sdk/actions/workflows/ci.yml)

A Go SDK for [OCPP](https://openchargealliance.org/protocols/open-charge-point-protocol/)
(Open Charge Point Protocol), the protocol between EV charging stations and their
central management system.

**Status: v1.** OCPP 1.6J (all 28 messages and the 11 of the security extension),
OCPP 2.0.1 (all 64 messages) and OCPP 2.1 (all 91 messages) are complete for both
roles, with WebSocket transport and security profiles 1 to 3. The public API follows
semantic versioning; see the [roadmap](AGENTS.md#7-roadmap) and the
[changelog](CHANGELOG.md).

- Both roles: charging station (`chargepoint`) and Central System / CSMS (`csms`).
- OCPP-J (JSON over WebSocket) only; no SOAP.
- Typed messages, validated when sent and when received, checked against the
  official JSON schemas in the test suite.
- Automatic reconnect with the back-off OCPP-J specifies; no goroutine or
  framing code to write.
- One dependency (`github.com/coder/websocket`), plus the standard library.

Documentation: <https://foecum.github.io/go-ocpp-sdk/> and the package docs on
[pkg.go.dev](https://pkg.go.dev/github.com/foecum/go-ocpp-sdk).

## Install

```bash
go get github.com/foecum/go-ocpp-sdk
```

## Quick start

### Central System

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/foecum/go-ocpp-sdk/csms/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// handler answers BootNotification. Embedding UnimplementedCoreHandler makes
// every other call from a charge point answer NotSupported.
type handler struct{ csms.UnimplementedCoreHandler }

func (handler) BootNotification(ctx context.Context, cp *csms.ChargePoint,
	req *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	log.Printf("%s booted: %s %s", cp.ID(), req.ChargePointVendor, req.ChargePointModel)
	return &core.BootNotificationConfirmation{
		Status:      types.RegistrationStatusAccepted,
		CurrentTime: types.Now(),
		Interval:    300,
	}, nil
}

func main() {
	srv, err := csms.New(csms.Config{Core: handler{}})
	if err != nil {
		log.Fatal(err)
	}
	// Charge points connect to ws://localhost:8080/ocpp/<identity>.
	http.Handle("/ocpp/", srv)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Charging station

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// handler answers Reset. Every other call from the Central System answers
// NotSupported.
type handler struct{ chargepoint.UnimplementedCoreHandler }

func (handler) Reset(ctx context.Context, req *core.ResetRequest) (*core.ResetConfirmation, error) {
	log.Println("reset requested:", req.Type)
	return &core.ResetConfirmation{Status: types.ResetStatusAccepted}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cp, err := chargepoint.New(chargepoint.Config{
		URL:  "ws://localhost:8080/ocpp",
		ID:   "CS001",
		Core: handler{},
	})
	if err != nil {
		log.Fatal(err)
	}
	go cp.Run(ctx) // connects, and reconnects by itself
	if err := cp.WaitConnected(ctx); err != nil {
		log.Fatal(err)
	}

	boot, err := cp.BootNotification(ctx, core.BootNotificationRequest{
		ChargePointVendor: "Acme",
		ChargePointModel:  "X1",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("boot:", boot.Status, "- heartbeat every", boot.Interval, "s")
	<-ctx.Done()
}
```

Run the first program, then the second: the Central System logs
`CS001 booted: Acme X1` and the station prints `boot: Accepted`.

### The Central System calls the station

A `*csms.ChargePoint` is a connected station. Look it up and call it with typed
methods; invalid requests are rejected before anything is sent:

```go
if cp, ok := srv.ChargePoint("CS001"); ok {
	conf, err := cp.Reset(ctx, core.ResetRequest{Type: types.ResetTypeSoft})
	// conf.Status is Accepted or Rejected
}
```

`srv.ChargePointIDs()` lists the connected stations, and `srv.ForEach(fn)` visits
each of them in identity order (return false from `fn` to stop).

Each OCPP profile has its own handler, set in the matching `Config` field
(`Core`, `Firmware`, `LocalAuth`, `Reservation`, `SmartCharging`,
`RemoteTrigger`, `Security`). A profile with no handler, or a method you left to the embedded
`Unimplemented...Handler`, is answered with `NotSupported`; an action the SDK
does not know is answered with `NotImplemented`.

## Supported messages

All 28 messages of OCPP 1.6 (JSON), and the 11 messages of its security extension:

| Profile | Messages |
|---|---|
| Core | Authorize, BootNotification, ChangeAvailability, ChangeConfiguration, ClearCache, DataTransfer, GetConfiguration, Heartbeat, MeterValues, RemoteStartTransaction, RemoteStopTransaction, Reset, StartTransaction, StatusNotification, StopTransaction, UnlockConnector |
| Firmware Management | GetDiagnostics, DiagnosticsStatusNotification, FirmwareStatusNotification, UpdateFirmware |
| Local Auth List Management | GetLocalListVersion, SendLocalList |
| Reservation | ReserveNow, CancelReservation |
| Smart Charging | SetChargingProfile, ClearChargingProfile, GetCompositeSchedule |
| Remote Trigger | TriggerMessage |
| Security extension | CertificateSigned, DeleteCertificate, ExtendedTriggerMessage, GetInstalledCertificateIds, GetLog, InstallCertificate, LogStatusNotification, SecurityEventNotification, SignCertificate, SignedFirmwareStatusNotification, SignedUpdateFirmware |

Message types are in `v16/core`, `v16/firmware`, `v16/localauth`,
`v16/reservation`, `v16/smartcharging`, `v16/remotetrigger` and `v16/security`;
shared types and enumerations are in `v16/types`. Each message is a `...Request` and a
`...Confirmation` type named after the specification.

## OCPP 2.0.1

All 64 messages are implemented, one package per functional block of the
specification (`v201/security`, `v201/provisioning`, `v201/authorization`,
`v201/transactions`, and so on; shared types in `v201/types`). They are generated
from the official JSON schemas and checked against them in the tests. The
high-level APIs are `csms/v201` and `chargepoint/v201` (package names `csms` and
`chargepoint`). They work like the 1.6 ones, with one handler interface per
functional block (`Config.Provisioning`, `Config.Authorization`,
`Config.Transactions`, ...) and an embeddable `Unimplemented...Handler` for each:

```go
type handler struct{ csms.UnimplementedProvisioningHandler }

func (handler) BootNotification(ctx context.Context, cp *csms.ChargePoint,
	req *provisioning.BootNotificationRequest) (*provisioning.BootNotificationResponse, error) {
	return &provisioning.BootNotificationResponse{
		Status:      types.RegistrationStatusAccepted,
		CurrentTime: ocpp.Now(),
		Interval:    300,
	}, nil
}

srv, _ := csms.New(csms.Config{Provisioning: handler{}})
```

The station side is `chargepoint.New(chargepoint.Config{URL: ..., ID: ...})`, then
`Run`, `WaitConnected` and typed calls such as `cp.BootNotification`. The Central
System calls a station with `srv.ChargePoint(id)` and methods such as
`GetVariables`, `SetVariables` and `RequestStartTransaction`. Security profiles,
logging and error handling are the same as for 1.6. The full program is on the
documentation site.

## OCPP 2.1

All 91 messages are implemented, in `v21` and one package per functional block like
2.0.1, plus the three blocks 2.1 adds: `v21/bidirectional` (Bidirectional Power
Transfer), `v21/dercontrol` (DER Control) and `v21/batteryswap` (Battery Swapping).
The high-level APIs are `csms/v21` and `chargepoint/v21`; they work like the 2.0.1
ones (the same `Config` fields for the old blocks, `DERControl` and `BatterySwap`
for the CSMS, plus `Bidirectional` for the station). The 27 messages new in 2.1:

| Block | New messages |
|---|---|
| Authorization | NotifySettlement, NotifyWebPaymentStarted, VatNumberValidation |
| Tariff and Cost | ChangeTransactionTariff, ClearTariffs, GetTariffs, SetDefaultTariff |
| Smart Charging | NotifyPriorityCharging, PullDynamicScheduleUpdate, UpdateDynamicSchedule, UsePriorityCharging |
| Certificate Management (`iso15118`) | GetCertificateChainStatus |
| Diagnostics | AdjustPeriodicEventStream, ClosePeriodicEventStream, GetPeriodicEventStream, NotifyPeriodicEventStream, OpenPeriodicEventStream |
| Bidirectional Power Transfer | AFRRSignal, NotifyAllowedEnergyTransfer |
| DER Control | ClearDERControl, GetDERControl, NotifyDERAlarm, NotifyDERStartStop, ReportDERControl, SetDERControl |
| Battery Swapping | BatterySwap, RequestBatterySwap |

`NotifyPeriodicEventStream` is an unconfirmed message: OCPP-J 2.1 sends it as a
SEND frame and the receiver never answers it. It has no `Request`/`Response`
pair. The station sends it with a method that returns only an error, and a CSMS
handler returns only an error too:

```go
err := cp.NotifyPeriodicEventStream(ctx, diagnostics.NotifyPeriodicEventStream{
	ID: 1, Basetime: ocpp.Now(),
	Data: []types.StreamDataElement{{T: 0, V: "230.4"}, {T: 5, V: "230.2"}},
})

func (h handler) NotifyPeriodicEventStream(ctx context.Context, cp *csms.ChargePoint,
	msg *diagnostics.NotifyPeriodicEventStream) error {
	return nil // messages arrive one at a time, in the order they were sent
}
```

The standardized names of the specification appendices (components, variables,
units, reason codes, security events, and for 2.1 idToken and connector types and
more) are constants in `v201/appendices` and `v21/appendices`, for example
`appendices.ComponentOCPPCommCtrlr`, `appendices.VariableHeartbeatInterval` and
`appendices.UnitKWh`. The messages carry them as plain strings, because a vendor
may add its own.

Besides the schemas, the messages enforce what the specification text adds, for
example that a Full `SendLocalList` carries an `idTokenInfo` for every entry, that
`evseId` is greater than 0 where the text says so, and, in 2.1, the V2X curve and
tariff-condition rules.

## One endpoint for every version

A charge point picks its OCPP version in the WebSocket handshake, so one URL can
serve a mixed fleet. Each version has its own CSMS; `ws.NewVersionRouter` sends
every handshake to the CSMS of the first version (in your order) that the station
offers:

```go
router, err := ws.NewVersionRouter(
	ws.VersionRoute{Version: ocpp.V21, Handler: srv21},  // *csms.CSMS from csms/v21
	ws.VersionRoute{Version: ocpp.V201, Handler: srv201}, // csms/v201
	ws.VersionRoute{Version: ocpp.V16, Handler: srv16},   // csms/v16
)
if err != nil {
	log.Fatal(err)
}
http.Handle("/ocpp/", router)
```

## Security profiles

| Profile | Central System | Charging station |
|---|---|---|
| 1: HTTP Basic auth over `ws://` | `Authenticator: ws.BasicAuthenticator(lookup)` | `Password` |
| 2: Basic auth over TLS | profile 1, plus `TLSConfig: ws.ServerTLSConfig(cert, nil)` | `Password`, `TLSConfig: ws.ClientTLSConfig(roots, nil)` |
| 3: TLS with client certificates | `TLSConfig: ws.ServerTLSConfig(cert, clientCAs)`, `Authenticator: ws.ClientCertAuthenticator(...)` | `TLSConfig: ws.ClientTLSConfig(roots, &clientCert)` |

TLS 1.2 is the minimum, and only the cipher suites OCPP 2.0.1 allows are enabled.
Profile 3 requires the certificate's common name to equal the charge point
identity. For example:

```go
cert, _ := tls.LoadX509KeyPair("csms.pem", "csms-key.pem")
clientCAs := x509.NewCertPool()
clientCAs.AppendCertsFromPEM(caPEM)
srv, _ := csms.New(csms.Config{
	Core:          handler{},
	TLSConfig:     ws.ServerTLSConfig(cert, clientCAs),
	Authenticator: ws.ClientCertAuthenticator(ws.ClientCertOptions{}),
})
log.Fatal(srv.ListenAndServe(":8443"))

// The station:
roots := x509.NewCertPool()
roots.AppendCertsFromPEM(caPEM)
clientCert, _ := tls.LoadX509KeyPair("cs001.pem", "cs001-key.pem")
cp, _ := chargepoint.New(chargepoint.Config{
	URL:       "wss://csms.example.com:8443/ocpp",
	ID:        "CS001",
	TLSConfig: ws.ClientTLSConfig(roots, &clientCert),
})
```

For OCPP 1.6 the Basic-auth password is the `AuthorizationKey`: 16 to 20 random
bytes written as 32 to 40 hexadecimal digits, sent as that text. `ws.BasicAuthenticator`
compares it as it is; `security.GenerateAuthorizationKey()` makes a suitable key and
`security.CheckAuthorizationKey` validates one. The certificate messages (`SignCertificate`,
`CertificateSigned`, `InstallCertificate` and so on) are in `v16/security`.

TLS 1.0 and 1.1, which the whitepaper tolerates for old stations that cannot do
TLS 1.2, are not supported: connections need TLS 1.2 or newer.

## Logging

Both `csms.Config` and `chargepoint.Config` take a `*slog.Logger`. The default
discards everything.

```go
logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
srv, _ := csms.New(csms.Config{Logger: logger})
```

## Error handling

Calls return errors you can branch on with `errors.Is` and `errors.As`:

| Error | Meaning |
|---|---|
| `ocpp.ErrNotConnected` | there is no connection, or it closed while the call waited for its answer |
| `ocpp.ErrTimeout` | no answer in time (`CallTimeout`, default 30 s) |
| `ocpp.ErrValidation` | the request was invalid and was not sent (`*ocpp.ValidationError` names the field) |
| `*ocpp.CallError` | the other side answered with a CALLERROR (`Code`, `Description`, `Details`) |
| `ocpp.ErrQueueFull`, `ocpp.ErrClosed` | too many calls waiting / the endpoint was closed |

```go
_, err := cp.Heartbeat(ctx, core.HeartbeatRequest{})
var callErr *ocpp.CallError
switch {
case err == nil:
case errors.Is(err, ocpp.ErrNotConnected):
	// offline: cp.WaitConnected(ctx), then decide whether to resend
case errors.Is(err, ocpp.ErrTimeout):
	// the other side may still have processed it
case errors.As(err, &callErr):
	log.Printf("answered %s: %s", callErr.Code, callErr.Description)
}
```

A handler can return a `*ocpp.CallError` to choose the CALLERROR sent to the peer.
Any other error, a nil result or a panic is answered with `InternalError`, and the
error text is never sent. Library code never panics on bad input.

### Offline behaviour

Calls made while disconnected fail at once. Nothing is queued or retried for you,
except through the opt-in `TransactionQueue` below. The station reconnects by
itself, with the back-off from OCPP-J section 5.4 (`Config.Reconnect`).

OCPP 1.6 asks stations to deliver StartTransaction, StopTransaction and MeterValues
in order, to keep them while offline and to retry what the Central System fails
to process (the `TransactionMessageAttempts` and `TransactionMessageRetryInterval`
configuration keys). `chargepoint/v16` has a `TransactionQueue` for that: hand it
those messages instead of calling the station, and run it.

```go
q, _ := chargepoint.NewTransactionQueue(cp, chargepoint.TransactionQueueConfig{
	Attempts:      3,                // TransactionMessageAttempts
	RetryInterval: 60 * time.Second, // TransactionMessageRetryInterval
	Store:         chargepoint.FileStore("transactions.json"), // optional: survives restarts
})
go q.Run(ctx)

tx, _ := q.StartTransaction(core.StartTransactionRequest{ConnectorID: 1, IDTag: "TAG", Timestamp: types.Now()})
// ... later, even while the station is offline:
_ = q.StopTransaction(tx, core.StopTransactionRequest{MeterStop: 1500, Timestamp: types.Now()})
```

The queue sends what it holds in order once the station is connected, fills in
the transactionId the Central System assigns, and drops a message after the last
failed attempt so one bad message cannot block the rest (`OnDiscard` reports it).

## Examples

Two runnable programs in [`examples/v16`](examples/v16) talk to each other. In
one terminal:

```bash
go run ./examples/v16/csms -addr :8080
```

In another:

```bash
go run ./examples/v16/chargepoint -id CS001
```

The station boots, sends heartbeats and runs charging sessions; the Central
System logs everything and calls the station back. Try `-password`, `-tags` and
`-sessions`. The test suite runs both programs against each other.

The same pair exists for OCPP 2.0.1 in [`examples/v201`](examples/v201):

```bash
go run ./examples/v201/csms -addr :8080 -remote-start
go run ./examples/v201/chargingstation -id CS001
```

The station runs sessions as `TransactionEvent`s (Started, Updated, Ended); the
Central System reads and sets a variable with `GetVariables` / `SetVariables`
and, with `-remote-start`, starts and stops a session remotely.

And for OCPP 2.1 in [`examples/v21`](examples/v21): the same session, plus a default
tariff set by the Central System and a periodic event stream sent as unconfirmed
messages.

```bash
go run ./examples/v21/csms -addr :8080
go run ./examples/v21/chargingstation -id CS001
```

### Load test

`go run ./examples/loadtest -stations 1000 -duration 30s` runs a CSMS and 1,000
simulated stations in one process (`-version 2.0.1` or `-version 2.1` for the other
versions) and prints
throughput and latency. Indicative benchmark results are in
[docs/benchmarks.md](docs/benchmarks.md).

## Packages

| Package | Purpose |
|---|---|
| `github.com/foecum/go-ocpp-sdk` | shared interfaces, errors and version constants |
| `.../csms/v16`, `.../chargepoint/v16` | the high-level APIs (package names `csms` and `chargepoint`) |
| `.../csms/v201`, `.../chargepoint/v201` | the high-level OCPP 2.0.1 APIs |
| `.../csms/v21`, `.../chargepoint/v21` | the high-level OCPP 2.1 APIs |
| `.../v16` and `.../v16/*` | OCPP 1.6 messages, types and registry |
| `.../v201` and `.../v201/*` | OCPP 2.0.1 messages (one package per functional block), types and registry |
| `.../v21` and `.../v21/*` | OCPP 2.1 messages (one package per functional block), types and registry |
| `.../v201/appendices`, `.../v21/appendices` | the standardized names of the specification appendices |
| `.../ocppj` | OCPP-J framing, call tracking and registries, independent of the transport |
| `.../ws` | the WebSocket transport and security helpers |

## Development

```bash
go build ./...
go vet ./...
go test -race ./...
scripts/coverage.sh   # fails if any package is below 95% coverage
```

Project scope, layout, conventions and the roadmap are in [AGENTS.md](AGENTS.md).
The OCPP specifications this SDK follows are listed in [docs/spec/README.md](docs/spec/README.md).

## Licence

[MIT](LICENSE). The OCPP specifications under `docs/spec/` are published by the
Open Charge Alliance and are not covered by this licence.
