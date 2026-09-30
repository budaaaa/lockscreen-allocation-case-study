package allocation_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"case-study-poc/allocation"
	"case-study-poc/mockpartner"
)

type Item = allocation.Item
type Source = allocation.Source
type MockPartner = mockpartner.MockPartner

func availableSource(name string, items ...Item) Source {
	return Source{
		Partner:  MockPartner{PartnerName: name, Items: items},
		Eligible: true,
	}
}

func TestHigherExpectedValueWins(t *testing.T) {
	expensive := Item{ID: "expensive", CPCMicros: 300_000, ClickPPM: 50_000}
	interesting := Item{ID: "interesting", CPCMicros: 100_000, ClickPPM: 300_000}
	sources := []Source{
		availableSource("high-cpc", expensive),
		availableSource("high-interest", interesting),
	}
	got, err := allocation.Allocate(context.Background(), 1, 50*time.Millisecond, sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "interesting" {
		t.Fatalf("want higher expected value item, got %+v", got.Items)
	}
}

func TestDeadlineKeepsFastResultAndFillsGap(t *testing.T) {
	fast := MockPartner{
		PartnerName: "fast",
		Delay:       2 * time.Millisecond,
		Items:       []Item{{ID: "fast-card", CPCMicros: 100_000, ClickPPM: 200_000}},
	}
	slow := MockPartner{
		PartnerName: "slow",
		Delay:       200 * time.Millisecond,
		Items:       []Item{{ID: "slow-card", CPCMicros: 500_000, ClickPPM: 200_000}},
	}
	sources := []Source{
		{Partner: fast, Eligible: true},
		{Partner: slow, Eligible: true},
	}
	fallback := []Item{{ID: "house-card", Partner: "fallback"}}
	start := time.Now()
	got, err := allocation.Allocate(context.Background(), 2, 20*time.Millisecond, sources, fallback)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) >= 100*time.Millisecond {
		t.Fatal("allocator waited for slow partner")
	}
	if len(got.Items) != 2 || got.Items[0].ID != "fast-card" || got.Items[1].ID != "house-card" {
		t.Fatalf("want fast result then fallback, got %+v", got.Items)
	}
	if !slices.Contains(got.TimedOut, "slow") {
		t.Fatalf("want slow partner timeout, got %+v", got.TimedOut)
	}
}

func TestPartnerFailureDoesNotLoseOtherResults(t *testing.T) {
	sources := []Source{
		{Partner: MockPartner{PartnerName: "broken", Fail: true}, Eligible: true},
		availableSource("healthy", Item{ID: "healthy-card", CPCMicros: 90_000, ClickPPM: 200_000}),
	}
	got, err := allocation.Allocate(context.Background(), 1, 50*time.Millisecond, sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "healthy-card" {
		t.Fatalf("want healthy partner item, got %+v", got.Items)
	}
	if got.Errors["broken"] == "" {
		t.Fatalf("want broken partner error, got %+v", got.Errors)
	}
}

type panicPartner struct{}

func (panicPartner) Name() string { return "ineligible" }
func (panicPartner) Fetch(context.Context, int) ([]Item, error) {
	panic("ineligible partner was contacted")
}

func TestIneligiblePartnerIsNotContacted(t *testing.T) {
	sources := []Source{
		{Partner: panicPartner{}, Eligible: false},
		availableSource("eligible", Item{ID: "card", CPCMicros: 100_000, ClickPPM: 100_000}),
	}
	got, err := allocation.Allocate(context.Background(), 1, 50*time.Millisecond, sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].Partner != "eligible" {
		t.Fatalf("want eligible result, got %+v", got.Items)
	}
}

func TestInvalidAndDuplicateCardsCannotFillTwice(t *testing.T) {
	sources := []Source{{Partner: MockPartner{PartnerName: "partner", Items: []Item{
		{ID: "same", CPCMicros: 100_000, ClickPPM: 100_000},
		{ID: "same", CPCMicros: 200_000, ClickPPM: 100_000},
		{ID: "invalid", CPCMicros: -1, ClickPPM: 100_000},
	}}, Eligible: true}}
	fallback := []Item{{ID: "house-one", Partner: "fallback"}, {ID: "house-two", Partner: "fallback"}}
	got, err := allocation.Allocate(context.Background(), 3, 50*time.Millisecond, sources, fallback)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 3 || got.Items[0].ID != "same" || got.Items[0].CPCMicros != 200_000 || got.Items[1].ID != "house-one" || got.Items[2].ID != "house-two" {
		t.Fatalf("want best duplicate and fallback, got %+v", got.Items)
	}
}

func TestInvalidRequestIsRejected(t *testing.T) {
	if _, err := allocation.Allocate(context.Background(), 0, time.Second, nil, nil); err == nil {
		t.Fatal("want error for zero items")
	}
	if _, err := allocation.Allocate(context.Background(), 1, 0, nil, nil); err == nil {
		t.Fatal("want error for zero budget")
	}
}

func TestParentCancellationReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := allocation.Allocate(ctx, 1, time.Second, nil, []Item{{ID: "house", Partner: "fallback"}})
	if err == nil {
		t.Fatal("cancelled caller should receive an error, not fallback cards")
	}
}

func TestDuplicatePartnerNamesAreRejected(t *testing.T) {
	sources := []Source{
		{Partner: MockPartner{PartnerName: "same"}, Eligible: true},
		{Partner: MockPartner{PartnerName: "same"}, Eligible: true},
	}
	if _, err := allocation.Allocate(context.Background(), 1, time.Second, sources, nil); err == nil {
		t.Fatal("duplicate partner names make outcomes ambiguous")
	}
}
