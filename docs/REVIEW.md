# Review of RFC-0042 and Interest Model v2

**Decision: request changes.** I would not approve either proposal as written. The allocator is the right direction, but this version risks mixing users' content, delaying the lockscreen, and reporting revenue we cannot trace. These are the blockers, in priority order.

## 1. Fix the personalised cache and missing-ID path

RFC section 3.4 caches partner responses by partner name, although the response can depend on GAID. That key could give one user's cards to another. Section 3.1 also fans out to every partner when no cohort matches and silently substitutes Device ID when GAID is missing. Section 4 logs full identifiers.

For launch, skip personalised response caching. Use a defined generic path for missing or deleted IDs, and log request or decision IDs with redacted diagnostics. Any substitute identifier or partner sharing needs product and privacy review. [Google Play guidance](https://support.google.com/googleplay/android-developer/answer/6048248/advertising-id?hl=en-GB) permits some fallback uses under conditions; the RFC should not assume one is always allowed.

## 2. Bound partner waiting and keep fallback

The RFC waits for every partner, with a five-second timeout for each. Parallel calls still finish at the speed of the slowest partner. Its rollout then removes the existing fallback in one release.

Give the whole allocation request a measured deadline. Use valid cards received by then, cancel late work, and fill gaps through the current fallback. Shadow the selector before ramping traffic. The app's latency target must set the deadline; five seconds is not a useful default for a lockscreen request.

## 3. Make clicks and payout traceable before using the model

Section 4 joins clicks to cards through GAID and nearby timestamps. Several cards can be returned together, so that join cannot reliably identify the clicked item. It also calculates old revenue using today's CPC. Give every returned card a serve ID and record its payout and position at serve time. Clicks reference that ID; record actual displays separately and count duplicate clicks once.

Interest Model v2 depends on these events. Its negatives cannot be linked reliably to returned items, it omits position, and it never defines whether the target is click per return or click after display. AUC rising from 0.74 to 0.81 does not show that probabilities are calibrated enough to multiply by CPC or that revenue improves. Defer v2 until the event contract, offline checks, and a small live test are in place.

## 4. Replace full cohorts instead of accumulating them

Section 3.2 says partners resend full GAID cohorts, but ingestion only adds IDs to Redis sets. Removed users would remain eligible. Validate each file, build a new version, then publish it as active. Define how long the last valid version can be used if a file is late or invalid.

With tens of millions of IDs per partner and about 20 partners expected, size storage before putting these sets on shared Redis. [Redis documents](https://redis.io/docs/latest/develop/data-types/sets/) the memory cost of large membership sets.

## Decisions needed before approval

- Which positions can join the auction? Keep internal, cached, SDK, and reserved content on existing paths unless product moves them.
- Can a partner fill more than one position? What category, freshness, or contractual rules apply?
- What does a portal override mean, and who can authorise it? An unrestricted always-win switch conflicts with the value goal.
- What is the payable rate for a programmatic item? A manually entered CPC may be stale, and a missing bid should not become a hardcoded 0.05.
- What are the latency, fill, cohort freshness, and launch thresholds?

## Keep and build on

An allocation service and parallel partner calls are useful foundations. Partner adapters can share a small interface. Commercial teams should be able to change validated rates and enablement without a deployment. BigQuery belongs in reporting and offline feature preparation, not on each request. A simple interest baseline is enough to start measuring.

Before v2 rollout, I would also require SQL and Go feature parity, repeatable training and artifact versions, and a safer fallback than `predicted_ctr = 1.0`. Those matter, but they do not repair the missing labels and attribution.
