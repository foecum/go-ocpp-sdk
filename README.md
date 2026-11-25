# go-ocpp-sdk

A Go SDK for [OCPP](https://openchargealliance.org/protocols/open-charge-point-protocol/)
(Open Charge Point Protocol), the protocol between EV charging stations and their
central management system.

**Status: pre-alpha.** Nothing is usable yet; the API will change.

## Goals

- Both roles: Charge Point / Charging Station (client) and Central System / CSMS (server).
- OCPP-J (JSON over WebSocket) only; no SOAP.
- Versions, in order: OCPP 1.6J, then OCPP 2.0.1, then OCPP 2.1 (later, optional).

## Install

```bash
go get github.com/foecum/go-ocpp-sdk
```

## Documentation

Usage docs: <https://foecum.github.io/go-ocpp-sdk/> (source in [`site/`](site/index.html)).

## Development

```bash
go build ./...
go vet ./...
go test -race ./...
```

Project scope, layout, conventions and the roadmap are in [AGENTS.md](AGENTS.md).
The OCPP specifications this SDK follows are listed in [docs/spec/README.md](docs/spec/README.md).

## Licence

[MIT](LICENSE). The OCPP specifications under `docs/spec/` are published by the
Open Charge Alliance and are not covered by this licence.
