// Package allocation selects partner cards under a request deadline.
package allocation

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

// ProbabilityScale is the number of probability units in one certainty.
const ProbabilityScale int64 = 1_000_000

// Item is a candidate card. CPCMicros is a dollar amount in millionths;
// ClickPPM is a probability in millionths.
type Item struct {
	ID        string `json:"id"`
	Partner   string `json:"partner,omitempty"`
	CPCMicros int64  `json:"cpc_micros"`
	ClickPPM  int64  `json:"click_ppm"`
}

// Partner supplies candidate items and must honor context cancellation.
type Partner interface {
	Name() string
	Fetch(context.Context, int) ([]Item, error)
}

// Source associates a partner with eligibility established by the caller.
type Source struct {
	Partner  Partner
	Eligible bool
}

// Result contains selected cards and request-level partner outcomes.
type Result struct {
	Items      []Item
	Candidates []Item
	TimedOut   []string
	Errors     map[string]string
	Elapsed    time.Duration
}

type partnerReply struct {
	index int
	items []Item
	err   error
}

type fetchOutcome struct {
	bySource [][]Item
	timedOut []string
	errors   map[string]string
}

// Allocate fetches eligible sources concurrently, ranks timely valid items,
// and fills any remaining positions from fallback.
func Allocate(ctx context.Context, n int, budget time.Duration, sources []Source, fallback []Item) (Result, error) {
	if n <= 0 || budget <= 0 {
		return Result{}, fmt.Errorf("item count and budget must be positive")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	names, err := validateSources(sources)
	if err != nil {
		return Result{}, err
	}

	start := time.Now()
	fetched := fetchPartners(ctx, n, budget, sources, names)
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	candidates := uniqueCandidates(fetched.bySource, names)
	return Result{
		Items:      selectWithFallback(candidates, fallback, n),
		Candidates: candidates,
		TimedOut:   fetched.timedOut,
		Errors:     fetched.errors,
		Elapsed:    time.Since(start),
	}, nil
}

func validateSources(sources []Source) ([]string, error) {
	names := make([]string, len(sources))
	seenNames := make(map[string]bool, len(sources))
	for index, source := range sources {
		if !source.Eligible {
			continue
		}
		if source.Partner == nil {
			return nil, fmt.Errorf("eligible partner is nil")
		}
		name := source.Partner.Name()
		if name == "" || seenNames[name] {
			return nil, fmt.Errorf("invalid or duplicate partner name %q", name)
		}
		names[index] = name
		seenNames[name] = true
	}
	return names, nil
}

func fetchPartners(ctx context.Context, n int, budget time.Duration, sources []Source, names []string) fetchOutcome {
	requestCtx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	deadline, _ := requestCtx.Deadline()
	outcome := fetchOutcome{
		bySource: make([][]Item, len(sources)),
		errors:   make(map[string]string),
	}
	replies := make(chan partnerReply, len(sources))
	responded := make([]bool, len(sources))
	pending := 0

	for index, source := range sources {
		if !source.Eligible {
			continue
		}
		pending++
		go func(index int, partner Partner) {
			items, err := partner.Fetch(requestCtx, n)
			replies <- partnerReply{index: index, items: items, err: err}
		}(index, source.Partner)
	}

collect:
	for pending > 0 {
		select {
		case reply := <-replies:
			receivedAt := time.Now()
			if responded[reply.index] {
				continue
			}
			responded[reply.index] = true
			pending--
			name := names[reply.index]
			if !replyWithinDeadline(receivedAt, deadline) || errors.Is(reply.err, context.DeadlineExceeded) || errors.Is(reply.err, context.Canceled) {
				outcome.timedOut = append(outcome.timedOut, name)
			} else if reply.err != nil {
				outcome.errors[name] = reply.err.Error()
			} else {
				outcome.bySource[reply.index] = reply.items
			}
		case <-requestCtx.Done():
			break collect
		}
	}
	for index, source := range sources {
		if source.Eligible && !responded[index] {
			outcome.timedOut = append(outcome.timedOut, names[index])
		}
	}
	return outcome
}

func uniqueCandidates(bySource [][]Item, names []string) []Item {
	var candidates []Item
	seen := make(map[string]int)
	for index, items := range bySource {
		for _, item := range items {
			if !Valid(item) {
				continue
			}
			item.Partner = names[index]
			if earlier, exists := seen[item.ID]; exists {
				if Value(item) > Value(candidates[earlier]) {
					candidates[earlier] = item
				}
				continue
			}
			seen[item.ID] = len(candidates)
			candidates = append(candidates, item)
		}
	}
	return candidates
}

func selectWithFallback(candidates, fallback []Item, n int) []Item {
	ranked := append([]Item(nil), candidates...)
	sort.Slice(ranked, func(i, j int) bool {
		if Value(ranked[i]) == Value(ranked[j]) {
			return ranked[i].ID < ranked[j].ID
		}
		return Value(ranked[i]) > Value(ranked[j])
	})
	selected := append([]Item(nil), ranked[:min(n, len(ranked))]...)
	seen := make(map[string]bool, len(candidates)+len(fallback))
	for _, item := range candidates {
		seen[item.ID] = true
	}
	for _, item := range fallback {
		if len(selected) == n {
			break
		}
		if item.ID == "" {
			continue
		}
		if seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		selected = append(selected, item)
	}
	return selected
}

// Valid reports whether an offered item has a usable ID, payout, and probability.
func Valid(item Item) bool {
	return item.ID != "" && item.CPCMicros > 0 && item.ClickPPM >= 0 && item.ClickPPM <= ProbabilityScale &&
		(item.ClickPPM == 0 || item.CPCMicros <= math.MaxInt64/item.ClickPPM)
}

// Value returns expected revenue in microdollars times probability units.
// The caller must use Valid for offered items before calling Value.
func Value(item Item) int64 {
	return item.CPCMicros * item.ClickPPM
}

func replyWithinDeadline(receivedAt, deadline time.Time) bool {
	return !receivedAt.After(deadline)
}
