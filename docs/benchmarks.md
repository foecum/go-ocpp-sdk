# Benchmarks

Indicative numbers, not guarantees. They were measured once on one machine and
vary with hardware, load and Go version; compare runs on the same machine only.
There are no pass or fail thresholds in CI.

- Machine: 8 logical CPUs, Linux 6.12 (amd64), Go 1.27.2
- Date: 2026-11-25

Run them yourself:

```bash
go test ./ocppj ./v16 ./v201 -run '^$' -bench . -benchmem
go test ./csms/v16 ./csms/v201 -run '^$' -bench Stations -benchtime 20000x
```

## Framing and decoding

| Benchmark | Time | Allocations |
|---|---|---|
| `ocppj` Parse a CALL frame | 1.2 µs | 12 |
| `ocppj` Marshal a CALL frame | 0.85 µs | 13 |
| `ocppj` decode and validate a small request | 1.5 µs | 10 |
| `ocppj` a complete call between two endpoints (no network) | 16.8 µs | 79 |
| 1.6 decode `BootNotification` | 4.1 µs | 21 |
| 1.6 decode `MeterValues` (3 sampled values) | 18 µs | 122 |
| 2.0.1 decode `BootNotification` | 5.0 µs | 33 |
| 2.0.1 decode `TransactionEvent` (2 sampled values) | 29 µs | 205 |

A real station sends a message every few seconds, so decoding is far from a
bottleneck. The required-field check walks the raw JSON a second time; if it ever
matters, that is the first place to look.

## Many stations on one CSMS

`BenchmarkStations` connects N simulated stations to one CSMS over loopback
WebSockets (both ends in the same process) and has all of them send heartbeats at
once. ns/op is the time per heartbeat round trip over all stations together, so
it is the inverse of throughput.

| Stations | 1.6 per call | 2.0.1 per call | Heap per station (both ends) | Connect all |
|---|---|---|---|---|
| 1 | 40 µs | 42 µs | 53 to 58 KB | under 1 ms |
| 100 | 8.2 µs | 8.4 µs | 37 to 41 KB | 4 to 6 ms |
| 1000 | 10.6 µs | 10.9 µs | 37 to 41 KB | 44 to 46 ms |

That is roughly 90,000 to 120,000 calls per second on this machine, with 1,000
connected stations. The heap figure counts the server side and the simulated
station together, so a CSMS alone needs about half of it. The one-station row is
slower per call because a single connection serializes its calls (OCPP allows
one outstanding CALL per direction).

## Load test

`go run ./examples/loadtest -stations 1000 -duration 10s -session-every 2s` runs a
CSMS and 1,000 simulated stations in one process (boot, a heartbeat per second,
and a charging session every 2 s, which is 4 messages in 1.6 and 4 in 2.0.1):

| Version | Connect all | Errors | p50 | p95 | p99 |
|---|---|---|---|---|---|
| 1.6 | 0.21 s | 0 | 98 µs | 8.0 ms | 12 ms |
| 2.0.1 | 0.21 s | 0 | 100 µs | 8.4 ms | 12 ms |

The tail latency is mostly scheduling: 2,000 busy sockets and goroutines on 8 CPUs
in a single process. 1,000 stations need about 2,000 file descriptors.
