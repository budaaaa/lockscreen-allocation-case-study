package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"case-study-poc/allocation"
)

const (
	fallbackPartner     = "fallback"
	scoreUnitsPerDollar = float64(allocation.ProbabilityScale) * 1_000_000
)

type runSettings struct {
	scenario string
	n        int
	budget   time.Duration
	runs     int
}

type measurements struct {
	sample          allocation.Result
	latencies       []time.Duration
	partnerCards    int
	selectedValue   int64
	fixedOrderValue int64
	cpcOnlyValue    int64
}

// measureScenario compares policies using only cards returned before the deadline.
func measureScenario(ctx context.Context, settings runSettings, sources []allocation.Source, order []string) (measurements, error) {
	measured := measurements{latencies: make([]time.Duration, 0, settings.runs)}
	for run := 0; run < settings.runs; run++ {
		fallback := fallbackCards(settings.n)
		result, err := allocation.Allocate(ctx, settings.n, settings.budget, sources, fallback)
		if err != nil {
			return measurements{}, err
		}
		if run == 0 {
			measured.sample = result
		}
		measured.latencies = append(measured.latencies, result.Elapsed)
		for _, item := range result.Items {
			measured.selectedValue += allocation.Value(item)
			if item.Partner != fallbackPartner {
				measured.partnerCards++
			}
		}
		for _, item := range fixedOrder(result.Candidates, fallback, settings.n, order) {
			measured.fixedOrderValue += allocation.Value(item)
		}
		for _, item := range cpcOrder(result.Candidates, fallback, settings.n) {
			measured.cpcOnlyValue += allocation.Value(item)
		}
	}
	sort.Slice(measured.latencies, func(i, j int) bool { return measured.latencies[i] < measured.latencies[j] })
	return measured, nil
}

func printMeasurements(out io.Writer, settings runSettings, measured measurements) {
	totalSlots := settings.n * settings.runs
	fmt.Fprintf(out, "scenario=%s n=%d budget=%s runs=%d\n", settings.scenario, settings.n, settings.budget, settings.runs)
	fmt.Fprintf(out, "latency p50=%s p95=%s\n",
		percentile(measured.latencies, 50).Round(time.Millisecond),
		percentile(measured.latencies, 95).Round(time.Millisecond))
	fmt.Fprintf(out, "partner fill=%.1f%%, fallback cards=%d of %d\n",
		100*float64(measured.partnerCards)/float64(totalSlots),
		totalSlots-measured.partnerCards, totalSlots)
	fmt.Fprintf(out, "expected value/request: allocator=$%.5f fixed-order=$%.5f cpc-only=$%.5f\n",
		dollarsPerRequest(measured.selectedValue, settings.runs),
		dollarsPerRequest(measured.fixedOrderValue, settings.runs),
		dollarsPerRequest(measured.cpcOnlyValue, settings.runs))
	fmt.Fprintln(out, "sample selection:")
	for _, item := range measured.sample.Items {
		fmt.Fprintf(out, "  %s (%s): expected $%.5f\n", item.ID, item.Partner, float64(allocation.Value(item))/scoreUnitsPerDollar)
	}
	if len(measured.sample.TimedOut) > 0 {
		fmt.Fprintf(out, "sample timed out: %v\n", measured.sample.TimedOut)
	}
	names := make([]string, 0, len(measured.sample.Errors))
	for name := range measured.sample.Errors {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(out, "sample error: %s: %s\n", name, measured.sample.Errors[name])
	}
}

func dollarsPerRequest(value int64, runs int) float64 {
	return float64(value) / scoreUnitsPerDollar / float64(runs)
}

func fallbackCards(n int) []allocation.Item {
	items := make([]allocation.Item, n)
	for i := range items {
		items[i] = allocation.Item{ID: fmt.Sprintf("house-%d", i+1), Partner: fallbackPartner}
	}
	return items
}

// These comparison policies use only candidates available to the allocator.
func fixedOrder(candidates, fallback []allocation.Item, n int, order []string) []allocation.Item {
	items := append([]allocation.Item(nil), candidates...)
	priority := make(map[string]int, len(order))
	for index, name := range order {
		priority[name] = index
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, leftOK := priority[items[i].Partner]
		right, rightOK := priority[items[j].Partner]
		if !leftOK {
			left = len(order)
		}
		if !rightOK {
			right = len(order)
		}
		return left < right
	})
	return fill(items, fallback, n)
}

func cpcOrder(candidates, fallback []allocation.Item, n int) []allocation.Item {
	items := append([]allocation.Item(nil), candidates...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CPCMicros == items[j].CPCMicros {
			return items[i].ID < items[j].ID
		}
		return items[i].CPCMicros > items[j].CPCMicros
	})
	return fill(items, fallback, n)
}

func fill(items, fallback []allocation.Item, n int) []allocation.Item {
	if len(items) >= n {
		return items[:n]
	}
	for _, item := range fallback {
		if len(items) == n {
			break
		}
		items = append(items, item)
	}
	return items
}

func percentile(sorted []time.Duration, p int) time.Duration {
	index := (p*len(sorted)+99)/100 - 1
	return sorted[index]
}
