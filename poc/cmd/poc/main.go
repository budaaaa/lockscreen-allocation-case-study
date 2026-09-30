package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	options, err := parseCLI(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		os.Exit(2)
	}
	config, err := loadConfig(options.configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	settings, err := options.settings(config.Defaults)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	sources, err := scenarioSources(config, settings.scenario)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	results, err := measureScenario(context.Background(), settings, sources, config.FixedOrder)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	printMeasurements(os.Stdout, settings, results)
}
