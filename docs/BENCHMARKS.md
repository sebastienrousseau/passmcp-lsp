<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Benchmarks

What analysis and hover cost. The server analyses a document on every
change, so an analysis has to be far below the interval between
keystrokes; a hover waits on passmcp, once per check id per session.

## Environment

Apple A18 Pro, macOS, Go 1.27.1, passmcp 0.0.1, measured on 2026-09-29 on
a machine running other builds at the same time: the load average was
about 42. The spread is wide for that reason, and the lower figures are the
better guide to an idle machine.

## Analysis, in process

`go test ./internal/check -run '^$' -bench . -benchtime 2s -count 3`:

| Benchmark | Input | Time per analysis (3 runs) | Allocations |
|---|---|---|---|
| `BenchmarkServerJSON` | `testdata/bench/server.json`, 1,066 bytes | 51.3 µs, 70.6 µs, 62.1 µs | 15.6 KB, 227 allocations |
| `BenchmarkPolicy` | `testdata/policy/accept/worked-example.json` | 16.3 µs, 17.0 µs, 17.8 µs | 6.3 KB, 106 allocations |
| `BenchmarkAttestation` | `testdata/attestation/mcp-statement.json` | 46.9 µs, 61.3 µs, 42.3 µs | 15.5 KB, 265 allocations |

## Processes

`hyperfine -N --warmup 3 --runs 30`:

| Command | Mean ± σ | Min … max |
|---|---|---|
| `passmcp-lsp check internal/check/testdata/bench/server.json` | 10.9 ms ± 2.9 ms | 7.4 ms … 20.0 ms |
| `passmcp explain --output json` on a one-finding report, with an empty `PASSMCP_CONFIG` (one hover lookup) | 69.5 ms ± 73.3 ms | 23.2 ms … 367.1 ms |

The first row is process start, one analysis and exit, which is what a CI
step pays per invocation. The second is the cost of the first hover on a
given check id; later hovers on it are answered from the cache.

## Reproducing

```sh
go test ./internal/check -run '^$' -bench . -benchtime 2s -count 3
make build
hyperfine -N --warmup 3 --runs 30 'build/passmcp-lsp check internal/check/testdata/bench/server.json'
```
