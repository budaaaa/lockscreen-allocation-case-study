package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"case-study-poc/allocation"
	"case-study-poc/mockpartner"
)

type Config struct {
	Defaults   Defaults                     `json:"defaults"`
	FixedOrder []string                     `json:"fixed_order"`
	Partners   []PartnerConfig              `json:"partners"`
	Scenarios  map[string]ScenarioOverrides `json:"scenarios"`
}

// ScenarioOverrides changes partner behavior without duplicating base card data.
type ScenarioOverrides map[string]PartnerOverride

type Defaults struct {
	N        int    `json:"n"`
	BudgetMS int    `json:"budget_ms"`
	Runs     int    `json:"runs"`
	Scenario string `json:"scenario"`
}

type PartnerConfig struct {
	Name     string            `json:"name"`
	DelayMS  int               `json:"delay_ms"`
	Eligible bool              `json:"eligible"`
	Fail     bool              `json:"fail"`
	Items    []allocation.Item `json:"items"`
}

type PartnerOverride struct {
	DelayMS  *int  `json:"delay_ms"`
	Eligible *bool `json:"eligible"`
	Fail     *bool `json:"fail"`
}

func loadConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Config{}, fmt.Errorf("config must contain one JSON object")
	}
	if err := config.validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (config Config) validate() error {
	if config.Defaults.N <= 0 || config.Defaults.BudgetMS <= 0 || config.Defaults.Runs <= 0 {
		return fmt.Errorf("default n, budget_ms, and runs must be positive")
	}
	if _, ok := config.Scenarios[config.Defaults.Scenario]; !ok {
		return fmt.Errorf("unknown default scenario %q", config.Defaults.Scenario)
	}
	names, err := config.validatePartners()
	if err != nil {
		return err
	}
	if err := config.validateFixedOrder(names); err != nil {
		return err
	}
	return config.validateScenarios(names)
}

func (config Config) validatePartners() (map[string]bool, error) {
	if len(config.Partners) < 3 {
		return nil, fmt.Errorf("proof of concept needs at least three partners")
	}
	names := make(map[string]bool, len(config.Partners))
	for _, partner := range config.Partners {
		if partner.Name == "" || names[partner.Name] || partner.DelayMS < 0 {
			return nil, fmt.Errorf("invalid or duplicate partner %q", partner.Name)
		}
		for _, item := range partner.Items {
			if !allocation.Valid(item) {
				return nil, fmt.Errorf("invalid item %q for partner %q", item.ID, partner.Name)
			}
		}
		names[partner.Name] = true
	}
	return names, nil
}

func (config Config) validateFixedOrder(names map[string]bool) error {
	if len(config.FixedOrder) != len(names) {
		return fmt.Errorf("fixed_order must list every partner once")
	}
	ordered := make(map[string]bool, len(config.FixedOrder))
	for _, name := range config.FixedOrder {
		if !names[name] || ordered[name] {
			return fmt.Errorf("invalid fixed_order partner %q", name)
		}
		ordered[name] = true
	}
	return nil
}

func (config Config) validateScenarios(names map[string]bool) error {
	for scenario, overrides := range config.Scenarios {
		for name, override := range overrides {
			if !names[name] || (override.DelayMS != nil && *override.DelayMS < 0) {
				return fmt.Errorf("invalid override %q in scenario %q", name, scenario)
			}
		}
	}
	return nil
}

func scenarioSources(config Config, scenario string) ([]allocation.Source, error) {
	overrides, ok := config.Scenarios[scenario]
	if !ok {
		return nil, fmt.Errorf("unknown scenario %q", scenario)
	}
	sources := make([]allocation.Source, 0, len(config.Partners))
	for _, partner := range config.Partners {
		if override, ok := overrides[partner.Name]; ok {
			if override.DelayMS != nil {
				partner.DelayMS = *override.DelayMS
			}
			if override.Fail != nil {
				partner.Fail = *override.Fail
			}
			if override.Eligible != nil {
				partner.Eligible = *override.Eligible
			}
		}
		sources = append(sources, allocation.Source{
			Eligible: partner.Eligible,
			Partner: mockpartner.MockPartner{
				PartnerName: partner.Name,
				Delay:       time.Duration(partner.DelayMS) * time.Millisecond,
				Fail:        partner.Fail,
				Items:       partner.Items,
			},
		})
	}
	return sources, nil
}
