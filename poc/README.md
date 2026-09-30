# Partner allocation POC

This local Go program asks three mock partners for cards, then chooses up to `N` cards before a request deadline. It demonstrates the two claims in [the design](../docs/DESIGN.md): a slow partner cannot hold the response, and a lower-CPC card can win when its expected click value is higher. It needs no credentials, services, or third-party packages.

## Run

Use Go 1.22 or newer from `poc/`:

```sh
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/poc -scenario=normal -n=3 -budget=80ms -runs=30
```

To reproduce the scenario table below, replace `normal` with `late`, `failure`, and `outage`. To reproduce the budget table, keep `normal` and try `10ms`, `40ms`, `80ms`, and `160ms`. Omitted flags use defaults from [scenarios.json](config/scenarios.json); supplied values override them. `-config` accepts another JSON file. Invalid input returns an error.

## How one request works

1. The [config loader](cmd/poc/config.go) reads partner cards and scenario overrides. Fast responds in 5 ms, premium in 45 ms, and flaky in 20 ms. Late scenarios delay premium to 150 ms; failure scenarios make flaky return an error.
2. The [allocator](allocation/allocator.go) calls eligible partners concurrently, keeps valid cards received by the deadline, removes duplicates, and ranks by `CPC × assumed click probability`. House cards fill gaps. The [mock adapter](mockpartner/mock_partner.go) simulates partner delay and failure.
3. The [measurement code](cmd/poc/measurement.go) reports latency, partner fill, and assumed value. Fixed-order and CPC-only comparisons see the same timely candidates. House cards have zero assumed click value.

## Measurements

Each row below is 30 sequential requests for three cards on a local macOS ARM64 machine with Go 1.26.8. Dollar figures are **assumed expected value**, not observed revenue.

| Scenario, 80 ms budget | Latency p50 / p95 | Partner fill | Allocator | Fixed order | CPC only |
| --- | ---: | ---: | ---: | ---: | ---: |
| Normal | 46 / 46 ms | 100% | $0.084 | $0.056 | $0.078 |
| Premium late | 81 / 81 ms | 100% | $0.056 | $0.056 | $0.056 |
| Flaky fails | 46 / 46 ms | 100% | $0.074 | $0.072 | $0.074 |
| Premium late, flaky fails | 81 / 81 ms | 66.7% | $0.032 | $0.032 | $0.032 |

In the normal case, `fast-video` wins over `premium-story`: its 8 cent CPC at a 25% assumed click chance is worth 2 cents per return; the story's 35 cent CPC at 4% is worth 1.4 cents.

| Normal scenario budget | Latency p50 / p95 | Partner fill | Allocator value |
| --- | ---: | ---: | ---: |
| 10 ms | 11 / 11 ms | 66.7% | $0.032 |
| 40 ms | 41 / 41 ms | 100% | $0.056 |
| 80 ms | 46 / 46 ms | 100% | $0.084 |
| 160 ms | 46 / 46 ms | 100% | $0.084 |

The 40 ms budget misses premium; 80 ms includes it. Raising the budget to 160 ms changes nothing here because every healthy mock partner has replied by about 46 ms. This curve illustrates the tradeoff, not a production latency target.

## Limits

Click probabilities are fixed assumptions. Positions are interchangeable; the POC does not solve position-specific assignment. Eligibility is an input flag, not a real GAID cohort lookup. It does not implement serve events, revenue reconciliation, caching, a portal, or a trained model. Production partner clients also need their own network timeouts and connection limits. The [review](../docs/REVIEW.md) and [design](../docs/DESIGN.md) cover those decisions.
