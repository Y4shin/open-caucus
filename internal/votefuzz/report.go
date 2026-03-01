package votefuzz

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadRegressionSeeds(path string) ([]uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file RegressionSeeds
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("unmarshal regression seeds: %w", err)
	}
	return file.Seeds, nil
}

func WriteFuzzReport(path string, report FuzzReport) error {
	data, err := yaml.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal fuzz report: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write fuzz report: %w", err)
	}
	return nil
}
