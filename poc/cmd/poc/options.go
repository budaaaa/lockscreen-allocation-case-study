package main

import (
	"flag"
	"fmt"
	"time"
)

const defaultConfigPath = "config/scenarios.json"

type cliOptions struct {
	configPath string
	itemCount  int
	budget     time.Duration
	scenario   string
	runs       int
	supplied   map[string]bool
}

func parseCLI(args []string) (cliOptions, error) {
	var options cliOptions
	flags := flag.NewFlagSet("poc", flag.ContinueOnError)
	flags.StringVar(&options.configPath, "config", defaultConfigPath, "scenario configuration file")
	flags.IntVar(&options.itemCount, "n", 0, "number of cards requested; defaults to config")
	flags.DurationVar(&options.budget, "budget", 0, "overall partner deadline; defaults to config")
	flags.StringVar(&options.scenario, "scenario", "", "scenario name; defaults to config")
	flags.IntVar(&options.runs, "runs", 0, "number of requests to measure; defaults to config")
	if err := flags.Parse(args); err != nil {
		return cliOptions{}, err
	}
	if flags.NArg() != 0 {
		err := fmt.Errorf("unexpected arguments: %v", flags.Args())
		fmt.Fprintln(flags.Output(), err)
		return cliOptions{}, err
	}
	options.supplied = make(map[string]bool)
	flags.Visit(func(selected *flag.Flag) {
		options.supplied[selected.Name] = true
	})
	return options, nil
}

func (options cliOptions) settings(defaults Defaults) (runSettings, error) {
	settings := runSettings{
		scenario: defaults.Scenario,
		n:        defaults.N,
		budget:   time.Duration(defaults.BudgetMS) * time.Millisecond,
		runs:     defaults.Runs,
	}
	if options.supplied["scenario"] {
		settings.scenario = options.scenario
	}
	if options.supplied["n"] {
		settings.n = options.itemCount
	}
	if options.supplied["budget"] {
		settings.budget = options.budget
	}
	if options.supplied["runs"] {
		settings.runs = options.runs
	}
	if settings.n <= 0 || settings.budget <= 0 || settings.runs <= 0 {
		return runSettings{}, fmt.Errorf("n, budget, and runs must be positive")
	}
	return settings, nil
}
