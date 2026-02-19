package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/px3-dev/keycloak-operator/internal/chart"
)

type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	manifest := flag.String("manifest", "", "path to upstream kubernetes.yml")
	output := flag.String("output", "chart", "output directory for Helm chart")
	var crds stringSlice
	flag.Var(&crds, "crd", "CRD file to include (repeatable)")
	flag.Parse()

	if *manifest == "" {
		fmt.Fprintln(os.Stderr, "error: --manifest is required")
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*manifest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading manifest: %v\n", err)
		os.Exit(1)
	}

	upstream, err := chart.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing manifest: %v\n", err)
		os.Exit(1)
	}

	if err := chart.Generate(upstream, *output, crds); err != nil {
		fmt.Fprintf(os.Stderr, "error generating chart: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated Helm chart for keycloak-operator %s in %s\n", upstream.AppVersion, *output)
}
