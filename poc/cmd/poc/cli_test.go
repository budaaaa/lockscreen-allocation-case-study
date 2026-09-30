package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExplicitNegativeItemCountIsRejected(t *testing.T) {
	goBinary := filepath.Join(runtime.GOROOT(), "bin", "go")
	command := exec.Command(goBinary, "run", ".", "-config=../../config/scenarios.json", "-n=-1")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("want an error for explicit -n=-1, got success:\n%s", output)
	}
	if !strings.Contains(string(output), "n, budget, and runs must be positive") {
		t.Fatalf("want a validation error, got:\n%s", output)
	}
}
