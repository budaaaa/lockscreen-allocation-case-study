package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"case-study-poc/allocation"
)

func caseConfig(t *testing.T) Config {
	t.Helper()
	config, err := loadConfig("../../config/scenarios.json")
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestOutageScenarioFillsWithFallback(t *testing.T) {
	sources, err := scenarioSources(caseConfig(t), "outage")
	if err != nil {
		t.Fatal(err)
	}
	got, err := allocation.Allocate(context.Background(), 3, 80*time.Millisecond, sources, fallbackCards(3))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 3 || got.Items[0].Partner != "fast" || got.Items[1].Partner != "fast" || got.Items[2].Partner != "fallback" {
		t.Fatalf("want two fast cards and one fallback, got %+v", got.Items)
	}
	if got.Errors["flaky"] == "" {
		t.Fatalf("want flaky failure reported, got %+v", got.Errors)
	}
}

func TestFixedOrderUsesSameAvailableCandidates(t *testing.T) {
	candidates := []allocation.Item{
		{ID: "premium", Partner: "premium", CPCMicros: 400_000, ClickPPM: 100_000},
		{ID: "flaky", Partner: "flaky", CPCMicros: 200_000, ClickPPM: 120_000},
		{ID: "fast", Partner: "fast", CPCMicros: 80_000, ClickPPM: 250_000},
	}
	got := fixedOrder(candidates, fallbackCards(3), 2, []string{"fast", "flaky", "premium"})
	if len(got) != 2 || got[0].ID != "fast" || got[1].ID != "flaky" {
		t.Fatalf("want static fast then flaky from available candidates, got %+v", got)
	}
}

func TestCPCOnlyComparisonCanChooseLowerValue(t *testing.T) {
	candidates := []allocation.Item{
		{ID: "premium-story", Partner: "premium", CPCMicros: 350_000, ClickPPM: 40_000},
		{ID: "fast-video", Partner: "fast", CPCMicros: 80_000, ClickPPM: 250_000},
	}
	got := cpcOrder(candidates, nil, 1)
	if len(got) != 1 || got[0].ID != "premium-story" {
		t.Fatalf("want CPC-only comparator to choose larger CPC, got %+v", got)
	}
}

func TestUnknownScenarioIsRejected(t *testing.T) {
	if _, err := scenarioSources(caseConfig(t), "unknown"); err == nil {
		t.Fatal("want unknown scenario error")
	}
}

func TestNormalScenarioCanPreferLowerCPC(t *testing.T) {
	sources, err := scenarioSources(caseConfig(t), "normal")
	if err != nil {
		t.Fatal(err)
	}
	got, err := allocation.Allocate(context.Background(), 3, 80*time.Millisecond, sources, fallbackCards(3))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 3 || got.Items[2].ID != "fast-video" {
		t.Fatalf("want lower-CPC fast video ahead of premium story, got %+v", got.Items)
	}
}

func TestConfigRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	invalidConfig := `{
		"defaults": {"n": 3, "budget_ms": 80, "runs": 1, "scenario": "normal"},
		"fixed_order": ["fast"],
		"partners": [],
		"scenarios": {"normal": {}},
		"surprise": true
	}`
	if err := os.WriteFile(path, []byte(invalidConfig), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(path); err == nil {
		t.Fatal("want an error for unknown configuration")
	}
}

func TestConfigRejectsInvalidItemRate(t *testing.T) {
	config := caseConfig(t)
	config.Partners[0].Items[0].ClickPPM = 1_000_001
	if err := config.validate(); err == nil {
		t.Fatal("want invalid click probability rejected before serving")
	}
}
