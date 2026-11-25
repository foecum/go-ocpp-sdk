# go-ocpp-sdk

[![CI](https://github.com/foecum/go-ocpp-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/foecum/go-ocpp-sdk/actions/workflows/ci.yml)

A Go SDK for [OCPP](https://openchargealliance.org/protocols/open-charge-point-protocol/)
(Open Charge Point Protocol), the protocol between EV charging stations and their
central management system.

**Status: pre-1.0.** OCPP 1.6J is complete: all 28 messages, both roles, WebSocket
transport with security profiles 1 to 3. The API may still change before 1.0.
OCPP 2.0.1 is next; see the [roadmap](AGENTS.md#7-roadmap).

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

Each OCPP profile has its own handler, set in the matching `Config` field
(`Core`, `Firmware`, `LocalAuth`, `Reservation`, `SmartCharging`,
`RemoteTrigger`). A profile with no handler, or a method you left to the embedded
`Unimplemented...Handler`, is answered with `NotSupported`; an action the SDK
does not know is answered with `NotImplemented`.

## Supported messages

All 28 messages of OCPP 1.6 (JSON):

| Profile | Messages |
|---|---|
| Core | Authorize, BootNotification, ChangeAvailability, ChangeConfiguration, ClearCache, DataTransfer, GetConfiguration, Heartbeat, MeterValues, RemoteStartTransaction, RemoteStopTransaction, Reset, StartTransaction, StatusNotification, StopTransaction, UnlockConnector |
| Firmware Management | GetDiagnostics, DiagnosticsStatusNotification, FirmwareStatusNotification, UpdateFirmware |
| Local Auth List Management | GetLocalListVersion, SendLocalList |
| Reservation | ReserveNow, CancelReservation |
| Smart Charging | SetChargingProfile, ClearChargingProfile, GetCompositeSchedule |
| Remote Trigger | TriggerMessage |

Message types are in `v16/core`, `v16/firmware`, `v16/localauth`,
`v16/reservation`, `v16/smartcharging` and `v16/remotetrigger`; shared types and
enumerations are in `v16/types`. Each message is a `...Request` and a
`...Confirmation` type named after the specification.

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

The OCPP 1.6 security whitepaper is not part of this repository's specifications,
so the profiles follow OCPP 2.0.1 part 2. The 1.6 Basic-auth password encoding
differs from 2.0.1 and is not handled.

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

Calls made while disconnected fail at once. Nothing is queued or retried for you
(OCPP 1.6 asks stations to retry transaction messages; build that on
`Connected` and `WaitConnected`). The station reconnects by itself, with the
back-off from OCPP-J section 5.4 (`Config.Reconnect`).

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

## Packages

| Package | Purpose |
|---|---|
| `github.com/foecum/go-ocpp-sdk` | shared interfaces, errors and version constants |
| `.../csms/v16`, `.../chargepoint/v16` | the high-level APIs (package names `csms` and `chargepoint`) |
| `.../v16` and `.../v16/*` | OCPP 1.6 messages, types and registry |
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
