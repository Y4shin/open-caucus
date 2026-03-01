package sqlite_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/Y4shin/conference-tool/internal/votefuzz"
)

func TestVoteFuzzRegressionSeeds(t *testing.T) {
	seeds, err := votefuzz.LoadRegressionSeeds("testdata/vote_fuzz_regressions.yaml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("vote fuzz regression seed file not found")
		}
		t.Fatalf("load regression seeds: %v", err)
	}
	if len(seeds) == 0 {
		t.Skip("no vote fuzz regression seeds configured")
	}

	for _, seed := range seeds {
		seed := seed
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			cfg, err := votefuzz.GenerateConfig(seed)
			if err != nil {
				t.Fatalf("generate config: %v", err)
			}
			res, err := votefuzz.ExecuteConfig(context.Background(), cfg, votefuzz.ExecuteOptions{})
			if err != nil {
				t.Fatalf("execute config: %v", err)
			}
			if !res.Passed {
				t.Fatalf("regression seed %d failed: %s (%v)", seed, res.StatusMessage, res.InvariantFailures)
			}
		})
	}
}
