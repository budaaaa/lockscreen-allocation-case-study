# Command center: lockscreen partner allocation

This is the map for the case study. Read the [ranked review](REVIEW.md) for the approval call, the [design](DESIGN.md) for the proposed system, the [POC README](../poc/README.md) for measured behavior, and [AI notes](AI_NOTES.md) for how the work was done. Research links are below.

**Status:** The written submission and local Go proof are packaged. Production cohort ingestion, events, and click prediction remain design proposals.

## The problem

The app has a few carousel positions and more partners competing for them. Static configuration picks partners without considering available cards, payable rate, user interest, or partner latency. A failed partner may leave a gap. We need to return up to **N suitable cards** quickly, explain each choice, and tie any later click to the card and payout actually served.

## The proposed solution

Auction only real-time positions that product opens to competition. Keep internal, cached, SDK, and reserved content on their existing paths.

1. **Eligibility:** Read a validated, versioned cohort snapshot. Missing or unmatched advertising IDs use an explicit generic path. Any substitute identifier needs product and privacy review.
2. **Deadline:** Fetch eligible partners concurrently under one measured request budget. Keep timely valid cards and fill gaps through the current fallback.
3. **Value:** Rank by payable CPC times the probability of a click after return at that position. Use item-level payout where available, conservative priors for sparse users, and explicit placement rules.
4. **Evidence:** Give each returned card a serve ID and payout snapshot. Record actual display separately; clicks reference the serve ID. Reconcile with partner settlement.
5. **Release:** Shadow the current selector, then ramp while watching latency, fill, errors, clicks, and realised revenue. Keep rollback available.

The [design](DESIGN.md) shows the production flow. The POC demonstrates deadline-bound fetching and value selection with local mocks; it does not claim to implement the full system.

## What the math can prove

Let `X(i,j)` be 1 when card `i` earns one billable click after return in position `j`, and 0 otherwise. Let `c(i)` be its payable CPC and `p(i,j) = P(X(i,j)=1)`. For a valid assignment `A`:

```text
E[revenue(A)] = E[sum over A of c(i) X(i,j)]
              = sum over A of c(i) p(i,j)
```

The second line uses linearity of expectation; clicks need not be independent. It assumes payout and each card's click chance do not change with the other cards selected. Under those assumptions, the best valid assignment maximises this sum.

For interchangeable positions, write `w(i) = c(i) p(i)`. If a chosen card `a` has `w(a) < w(b)` for an unchosen card `b`, swapping them raises expected revenue by `w(b) - w(a) > 0`. Repeating that swap proves top N is optimal. With position rules or position-specific probabilities, solve the small assignment problem instead. For example, `$0.10 × 0.30 = $0.030`, which beats `$0.30 × 0.05 = $0.015` despite the lower CPC.

The latency claim is separate: `T(wait for all) = max T(partner)`, while a cancellable request with budget `B` stops partner waiting by `B`, plus local work. The [budget sweep](../poc/README.md#measurements) shows the value lost at 10 and 40 ms and gained at 80 ms. It does not choose a production deadline; that needs app latency and real revenue data.

Two other checks explain the data decisions. If a partner sends full cohorts, `old ∪ new` keeps removed IDs, so publish `new` as a replacement. And AUC alone cannot approve the model: squaring probabilities preserves ranking and AUC, yet can reverse CPC-weighted choices. At `$1 × 0.20` versus `$2 × 0.11`, the second wins (`$0.22 > $0.20`); after squaring, the first wins (`$0.04 > $0.0242`). The policy is proven under its assumptions, not the accuracy of our estimates.

## Decision log

“Design” means proposed for production. “POC” means demonstrated locally.

| ID | Decision | Evidence |
| --- | --- | --- |
| D1 | Auction only opened real-time positions; leave other content paths alone. | Design; [scope](DESIGN.md) |
| D2 | One overall deadline, partial results, and existing fallback protect response time. | Design and POC; [results](../poc/README.md) |
| D3 | Compare payable CPC times click chance. Top N requires interchangeable positions. | Design and POC; [math above](#what-the-math-can-prove) |
| D4 | Do not cache personalised partner responses by partner alone. | Design; [review](REVIEW.md) |
| D5 | Replace each validated full cohort snapshot instead of unioning it. | Design; [review](REVIEW.md) |
| D6 | Join clicks by serve ID and retain payout at serve time; track display separately. | Design; [design](DESIGN.md) |
| D7 | Use a generic path for missing GAID; any identifier substitution needs review. | Design; [Google Play guidance](https://support.google.com/googleplay/android-developer/answer/6048248/advertising-id?hl=en-GB) |
| D8 | Start with conservative priors; defer Interest Model v2 until labels, calibration, and live evidence exist. | Design; [appendix review](REVIEW.md) |
| D9 | Keep mock partner data in JSON and the allocator independent of it. | POC; [mock data](../poc/config/scenarios.json) |
| D10 | Keep focused test inputs beside assertions and separate fetching, selection, and measurement in code. | POC; [walkthrough](../poc/README.md) |

## Questions that change the design

- What latency budget and number of auctionable positions does the app need?
- Can one partner fill several positions, and what placement or commercial rules apply?
- How fresh must cohorts be, and are partner files always full replacements?
- When is CPC final: in a returned item, at serve time, or at settlement?
- Which identifier and event-sharing rules have product and privacy owners approved?

## Work completed in order

This records the visible collaboration. AI drafted and coded; the human directed scope, challenged assumptions, and requested changes. It does not claim a production build.

| Step | Task | Evidence |
| --- | --- | --- |
| 1 | Read the three PDFs and wrote the problem statement. | [Sources](#research-notes), [problem](#the-problem) |
| 2 | Checked policy, BigQuery, Redis, and click-bias claims against primary sources. | [Research notes](#research-notes) |
| 3 | Wrote the value and deadline equations; corrected an overstatement about unviewed cards. | [Math](#what-the-math-can-prove), [AI notes](AI_NOTES.md) |
| 4 | Ranked RFC and model findings; wrote the production design and deferrals. | [Review](REVIEW.md), [design](DESIGN.md) |
| 5 | Wrote allocator tests, then built three mocks and the Go POC. | [Tests](../poc/allocation/allocator_test.go), [code](../poc/allocation/allocator.go) |
| 6 | Measured normal, late, failure, and outage cases; fixed a deadline edge and weak CPC example found in review. | [Results](../poc/README.md), [test](../poc/allocation/deadline_test.go) |
| 7 | Split reusable allocation, mock I/O, configuration, and command responsibilities. | [Walkthrough](../poc/README.md) |
| 8 | Ran tests, race checks, vet, scenarios, and a fresh-zip check. | [Verification route](#agent-review-route) |
| 9 | Improved code and document readability; added the two design diagrams. | [Design](DESIGN.md), [AI notes](AI_NOTES.md) |
| 10 | Audited the exercise, sources, links, and package contents. | [Review](REVIEW.md) |
| 11 | Removed CLI sentinel defaults and added a regression test for explicit `-n=-1`. | [Options](../poc/cmd/poc/options.go), [test](../poc/cmd/poc/cli_test.go) |
| 12 | Ran a four-budget sweep and edited the submission for clarity. | [Measurements](../poc/README.md) |

## Agent review route

1. Read the original exercise instructions, RFC-0042, and Interest Model v2 from the separately supplied source pack if available. Then read the [review](REVIEW.md), [design](DESIGN.md), and [AI notes](AI_NOTES.md).
2. Check the POC's claims and limits in its [README](../poc/README.md). From `poc/`, run `go test ./...`, `go test -race ./...`, and `go vet ./...`.
3. Run `go run ./cmd/poc -scenario=normal -n=3 -budget=80ms -runs=30`; repeat with `late`, `failure`, and `outage`, or change `-budget` to `10ms`, `40ms`, and `160ms`. Compare with the README tables.

## Research notes

The original exercise instructions, RFC-0042, and Interest Model v2 are the source of truth. They were supplied separately and are not included here. The external sources support specific risks, not the whole architecture.

- [Google Play advertising ID guidance](https://support.google.com/googleplay/android-developer/answer/6048248/advertising-id?hl=en-GB): reset, deletion, and conditional fallback rules.
- [Google Play user data policy](https://support.google.com/googleplay/android-developer/answer/10144311?hl=en-GB): identifier linking, disclosure, and sharing.
- [BigQuery reliability guidance](https://docs.cloud.google.com/bigquery/docs/reliability-intro): analytics focus and alternatives for transactional workloads.
- [Redis sets](https://redis.io/docs/latest/develop/data-types/sets/): membership checks and memory cost at scale.
- [Google Research on click bias](https://research.google/pubs/towards-disentangling-relevance-and-bias-in-unbiased-learning-to-rank/): position effects in click data. Serve IDs and controlled evaluation are our inference from this and the RFC.
