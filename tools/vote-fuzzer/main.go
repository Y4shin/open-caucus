package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Y4shin/open-caucus/internal/votefuzz"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "vote-fuzzer",
		Short: "Seeded vote fuzzing utility",
	}
	root.AddCommand(newFuzzCmd())
	root.AddCommand(newExplainCmd())
	root.AddCommand(newInspectCmd())
	return root
}

func newFuzzCmd() *cobra.Command {
	var (
		initialSeed uint64
		count       int
		jobs        int
		outputPath  string
	)

	cmd := &cobra.Command{
		Use:   "fuzz",
		Short: "Run seeded vote fuzz cases",
		RunE: func(cmd *cobra.Command, args []string) error {
			if count <= 0 {
				return fmt.Errorf("--count must be > 0")
			}
			if jobs <= 0 {
				return fmt.Errorf("--jobs must be > 0")
			}
			ctx := cmd.Context()
			start := time.Now()

			type job struct {
				seed uint64
			}
			type jobResult struct {
				seed   uint64
				result *votefuzz.ExecutionResult
				err    error
			}

			jobsCh := make(chan job, jobs*2)
			resultsCh := make(chan jobResult, jobs*2)

			var wg sync.WaitGroup
			for worker := 0; worker < jobs; worker++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for item := range jobsCh {
						cfg, err := votefuzz.GenerateConfig(item.seed)
						if err != nil {
							resultsCh <- jobResult{seed: item.seed, err: fmt.Errorf("generate config: %w", err)}
							continue
						}
						res, err := votefuzz.ExecuteConfig(ctx, cfg, votefuzz.ExecuteOptions{})
						if err != nil {
							resultsCh <- jobResult{seed: item.seed, err: err}
							continue
						}
						resultsCh <- jobResult{seed: item.seed, result: &res}
					}
				}()
			}

			go func() {
				for i := 0; i < count; i++ {
					seed := votefuzz.DeriveSeed(initialSeed, "fuzz-case", i)
					jobsCh <- job{seed: seed}
				}
				close(jobsCh)
			}()

			go func() {
				wg.Wait()
				close(resultsCh)
			}()

			bar := progressbar.NewOptions(
				count,
				progressbar.OptionSetDescription("fuzz"),
				progressbar.OptionShowCount(),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionUseANSICodes(true),
				progressbar.OptionThrottle(120*time.Millisecond),
				progressbar.OptionSetRenderBlankState(true),
				progressbar.OptionShowElapsedTimeOnFinish(),
			)

			report := votefuzz.FuzzReport{
				Invocation: votefuzz.FuzzInvocation{
					InitialSeed: initialSeed,
					Count:       count,
					Jobs:        jobs,
					Timestamp:   time.Now().UTC(),
				},
				Failures: make([]votefuzz.FuzzFailure, 0),
			}

			for result := range resultsCh {
				_ = bar.Add(1)
				if result.err != nil {
					report.Failures = append(report.Failures, votefuzz.FuzzFailure{
						Seed:    result.seed,
						Status:  "error",
						Message: result.err.Error(),
					})
					continue
				}
				if result.result != nil && result.result.Passed {
					continue
				}
				message := "failed"
				if result.result != nil && result.result.StatusMessage != "" {
					message = result.result.StatusMessage
				}
				report.Failures = append(report.Failures, votefuzz.FuzzFailure{
					Seed:    result.seed,
					Status:  "failed",
					Message: message,
				})
			}
			_ = bar.Finish()

			report.Summary = votefuzz.FuzzSummary{
				Passed:     count - len(report.Failures),
				Failed:     len(report.Failures),
				DurationMS: time.Since(start).Milliseconds(),
			}

			if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
				return fmt.Errorf("create output directory: %w", err)
			}
			if err := votefuzz.WriteFuzzReport(outputPath, report); err != nil {
				return err
			}
			cmd.Printf("wrote report: %s\n", outputPath)
			return nil
		},
	}

	cmd.Flags().Uint64Var(&initialSeed, "initial-seed", 1, "Initial seed")
	cmd.Flags().IntVar(&count, "count", 100, "Number of generated test configs")
	cmd.Flags().IntVar(&jobs, "jobs", 4, "Parallel worker count")
	cmd.Flags().StringVar(&outputPath, "out", "vote-fuzz-report.yaml", "Output YAML path")
	return cmd
}

func newExplainCmd() *cobra.Command {
	var seed uint64
	cmd := &cobra.Command{
		Use:   "explain",
		Short: "Explain one generated config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := votefuzz.ExplainConfig(seed)
			if err != nil {
				return err
			}
			enc := yaml.NewEncoder(os.Stdout)
			enc.SetIndent(2)
			defer enc.Close()
			return enc.Encode(cfg)
		},
	}
	cmd.Flags().Uint64Var(&seed, "seed", 0, "Config seed")
	_ = cmd.MarkFlagRequired("seed")
	return cmd
}

func newInspectCmd() *cobra.Command {
	var seed uint64
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Execute one seed and print detailed diagnostics",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := votefuzz.InspectSeed(context.Background(), seed, votefuzz.ExecuteOptions{})
			if err != nil {
				return err
			}
			enc := yaml.NewEncoder(os.Stdout)
			enc.SetIndent(2)
			defer enc.Close()
			if err := enc.Encode(res); err != nil {
				return err
			}
			if !res.Passed {
				return fmt.Errorf("seed %d produced invalid state: %s", seed, res.StatusMessage)
			}
			return nil
		},
	}
	cmd.Flags().Uint64Var(&seed, "seed", 0, "Config seed")
	_ = cmd.MarkFlagRequired("seed")
	return cmd
}
