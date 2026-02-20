package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Y4shin/conference-tool/tools/routing"
)

func main() {
	var (
		configFile      = flag.String("config", "routes.yaml", "Path to the routes configuration YAML file")
		outputFile      = flag.String("output", "routes_gen.go", "Path to the generated routes output file")
		packageName     = flag.String("package", "routes", "Package name for the generated routes code")
		pathsOutputFile = flag.String("paths-output", "", "Path to the generated paths output file (if empty, paths are not generated separately)")
		pathsPackage    = flag.String("paths-package", "paths", "Package name for the generated paths code")
		staticDir       = flag.String("static-dir", "", "Path to static assets directory to embed (auto-detected as 'static/' next to output file if not specified)")
		localePackage   = flag.String("locale-package", "", "Import path of the locale package (enables locale-aware path builders)")
	)

	flag.Parse()

	if err := run(*configFile, *outputFile, *packageName, *pathsOutputFile, *pathsPackage, *staticDir, *localePackage); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(configFile, outputFile, packageName, pathsOutputFile, pathsPackage, staticDirFlag, localePackage string) error {
	// Parse the configuration
	config, err := routing.ParseConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Resolve static directory
	staticDir := staticDirFlag
	if staticDir == "" {
		staticDir = filepath.Join(filepath.Dir(outputFile), "static")
	}

	// Scan for static files (returns nil, nil when directory doesn't exist)
	staticFiles, err := routing.ScanStaticDir(staticDir)
	if err != nil {
		return fmt.Errorf("failed to scan static directory %q: %w", staticDir, err)
	}

	// Generate the routes code
	code, err := routing.Generate(config, packageName, staticFiles)
	if err != nil {
		return fmt.Errorf("failed to generate routes code: %w", err)
	}

	// Write routes output file
	if err := os.WriteFile(outputFile, []byte(code), 0644); err != nil {
		return fmt.Errorf("failed to write routes output file: %w", err)
	}
	fmt.Printf("Successfully generated %s from %s\n", outputFile, configFile)

	// Generate paths if output specified
	if pathsOutputFile != "" {
		pathsCode, err := routing.GeneratePaths(config, pathsPackage, staticFiles, localePackage)
		if err != nil {
			return fmt.Errorf("failed to generate paths code: %w", err)
		}

		// Ensure paths directory exists
		pathsDir := filepath.Dir(pathsOutputFile)
		if err := os.MkdirAll(pathsDir, 0755); err != nil {
			return fmt.Errorf("failed to create paths directory: %w", err)
		}

		if err := os.WriteFile(pathsOutputFile, []byte(pathsCode), 0644); err != nil {
			return fmt.Errorf("failed to write paths output file: %w", err)
		}
		fmt.Printf("Successfully generated %s from %s\n", pathsOutputFile, configFile)
	}

	return nil
}
