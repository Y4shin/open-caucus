package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Y4shin/conference-tool/tools/routing"
)

func main() {
	var (
		configFile  = flag.String("config", "routes.yaml", "Path to the routes configuration YAML file")
		outputFile  = flag.String("output", "routes_gen.go", "Path to the generated output file")
		packageName = flag.String("package", "routes", "Package name for the generated code")
	)

	flag.Parse()

	if err := run(*configFile, *outputFile, *packageName); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated %s from %s\n", *outputFile, *configFile)
}

func run(configFile, outputFile, packageName string) error {
	// Parse the configuration
	config, err := routing.ParseConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Generate the code
	code, err := routing.Generate(config, packageName)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	// Write to output file
	if err := os.WriteFile(outputFile, []byte(code), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
