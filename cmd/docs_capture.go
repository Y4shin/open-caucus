package cmd

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Y4shin/open-caucus/tools/docscapture"
	"github.com/Y4shin/open-caucus/tools/docscapture/scripts"
	"github.com/spf13/cobra"
)

func newDocsCaptureCmd() *cobra.Command {
	var (
		outDir         string
		viewportWidth  int
		viewportHeight int
		headed         bool
		slowMoMS       float64
	)

	cmd := &cobra.Command{
		Use:   "docs-capture",
		Short: "Capture screenshots and GIFs for internal documentation",
		Long: `Capture deterministic visual artifacts in a self-hosted, per-script test environment.
Capture scripts are selected by glob and executed for one or more theme/language/device variants.`,
	}

	cmd.PersistentFlags().StringVar(&outDir, "out-dir", "doc/assets/captures", "Output directory for generated artifacts")
	cmd.PersistentFlags().IntVar(&viewportWidth, "viewport-width", 1440, "Browser viewport width in pixels")
	cmd.PersistentFlags().IntVar(&viewportHeight, "viewport-height", 900, "Browser viewport height in pixels")
	cmd.PersistentFlags().BoolVar(&headed, "headed", false, "Run Chromium in headed mode")
	cmd.PersistentFlags().Float64Var(&slowMoMS, "slow-mo-ms", 0, "Slow down each Playwright operation by N milliseconds")

	commonOptions := func() docscapture.CommonOptions {
		return docscapture.CommonOptions{
			BaseURL:        "",
			OutDir:         outDir,
			ViewportWidth:  viewportWidth,
			ViewportHeight: viewportHeight,
			Headed:         headed,
			SlowMoMS:       slowMoMS,
		}
	}

	cmd.AddCommand(newDocsCaptureListCmd())
	cmd.AddCommand(newDocsCaptureRunCmd(commonOptions))
	return cmd
}

func newDocsCaptureListCmd() *cobra.Command {
	var globs []string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available capture scripts",
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := scripts.Match(globs)
			if err != nil {
				return err
			}
			if len(selected) == 0 {
				return fmt.Errorf("no scripts match globs: %v", globs)
			}

			for _, script := range selected {
				cmd.Printf("%s\t%s\n", script.Name, script.Description)
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&globs, "script", []string{"*"}, "Script name globs to include (repeatable)")
	return cmd
}

func newDocsCaptureRunCmd(commonOptions func() docscapture.CommonOptions) *cobra.Command {
	var (
		scriptGlobs []string
		rawThemes   []string
		rawLangs    []string
		rawDevices  []string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run capture scripts selected by glob",
		Long: `Run one or more capture scripts in isolated self-hosted environments.
Each selected script is executed for each selected (theme, language, device) variant.`,
		Example: fmt.Sprintf(
			`  %s docs-capture run --script "app.screenshot-*" --theme light --language en --device desktop
  %s docs-capture run --script "app.gif-*" --theme light,dark --language english,german --device desktop,mobile`,
			rootCmd.Use,
			rootCmd.Use,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := scripts.Match(scriptGlobs)
			if err != nil {
				return err
			}
			if len(selected) == 0 {
				return fmt.Errorf("no scripts match globs: %v", scriptGlobs)
			}

			themes, err := parseThemes(rawThemes)
			if err != nil {
				return err
			}
			langs, err := parseLanguages(rawLangs)
			if err != nil {
				return err
			}
			devices, err := parseDevices(rawDevices)
			if err != nil {
				return err
			}
			variants := cartesianVariants(themes, langs, devices)
			if len(variants) == 0 {
				return fmt.Errorf("no variants selected")
			}

			type captureJob struct {
				script  scripts.Script
				variant docscapture.Variant
			}

			jobs := make([]captureJob, 0, len(selected)*len(variants))
			for _, script := range selected {
				for _, variant := range variants {
					jobs = append(jobs, captureJob{
						script:  script,
						variant: variant,
					})
				}
			}

			total := len(jobs)
			cmd.Printf("Running %d docs-capture jobs\n", total)

			for i, job := range jobs {
				cmd.Printf(
					"%s %d/%d START %-32s language=%-2s theme=%-5s device=%-7s\n",
					renderProgressBar(i, total, 24),
					i+1,
					total,
					job.script.Name,
					job.variant.Language,
					job.variant.Theme,
					job.variant.Device,
				)
				startedAt := time.Now()

				script := job.script
				variant := job.variant

				env, err := docscapture.NewEnvironment(script.Environment)
				if err != nil {
					return fmt.Errorf(
						"start self-hosted environment for script %q (language=%s theme=%s device=%s): %w",
						script.Name,
						variant.Language,
						variant.Theme,
						variant.Device,
						err,
					)
				}

				common := commonOptions()
				common.BaseURL = env.BaseURL()

				outPath, runErr := script.Run(cmd.Context(), common, variant, env)
				closeErr := env.Close()
				if runErr != nil {
					if closeErr != nil {
						runErr = errors.Join(runErr, fmt.Errorf("close environment: %w", closeErr))
					}
					return fmt.Errorf(
						"script %q failed for language=%s theme=%s device=%s: %w",
						script.Name,
						variant.Language,
						variant.Theme,
						variant.Device,
						runErr,
					)
				}
				if closeErr != nil {
					return fmt.Errorf(
						"close self-hosted environment for script %q (language=%s theme=%s device=%s): %w",
						script.Name,
						variant.Language,
						variant.Theme,
						variant.Device,
						closeErr,
					)
				}
				cmd.Printf(
					"%s %d/%d DONE  %-32s language=%-2s theme=%-5s device=%-7s (%s) -> %s\n",
					renderProgressBar(i+1, total, 24),
					i+1,
					total,
					script.Name,
					variant.Language,
					variant.Theme,
					variant.Device,
					time.Since(startedAt).Round(10*time.Millisecond),
					outPath,
				)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&scriptGlobs, "script", []string{"*"}, "Script name globs to include (repeatable)")
	cmd.Flags().StringSliceVar(&rawThemes, "theme", []string{"light", "dark"}, "Theme variants: light,dark (aliases: lightmode,darkmode)")
	cmd.Flags().StringSliceVar(&rawLangs, "language", []string{"en", "de"}, "Language variants: en,de (aliases: english,german,deutsch)")
	cmd.Flags().StringSliceVar(&rawDevices, "device", []string{"desktop", "mobile"}, "Device variants: desktop,mobile")
	return cmd
}

func parseThemes(raw []string) ([]docscapture.Theme, error) {
	parsed := make([]docscapture.Theme, 0, len(raw))
	for _, value := range raw {
		theme, err := docscapture.ParseTheme(value)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(parsed, theme) {
			parsed = append(parsed, theme)
		}
	}
	return parsed, nil
}

func parseLanguages(raw []string) ([]docscapture.Language, error) {
	parsed := make([]docscapture.Language, 0, len(raw))
	for _, value := range raw {
		lang, err := docscapture.ParseLanguage(value)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(parsed, lang) {
			parsed = append(parsed, lang)
		}
	}
	return parsed, nil
}

func parseDevices(raw []string) ([]docscapture.Device, error) {
	parsed := make([]docscapture.Device, 0, len(raw))
	for _, value := range raw {
		device, err := docscapture.ParseDevice(value)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(parsed, device) {
			parsed = append(parsed, device)
		}
	}
	return parsed, nil
}

func cartesianVariants(themes []docscapture.Theme, languages []docscapture.Language, devices []docscapture.Device) []docscapture.Variant {
	variants := make([]docscapture.Variant, 0, len(themes)*len(languages)*len(devices))
	for _, language := range languages {
		for _, theme := range themes {
			for _, device := range devices {
				variants = append(variants, docscapture.Variant{
					Theme:    theme,
					Language: language,
					Device:   device,
				})
			}
		}
	}
	return variants
}

func renderProgressBar(done, total, width int) string {
	if total <= 0 {
		return "[" + strings.Repeat("-", max(width, 1)) + "]"
	}
	if width <= 0 {
		width = 1
	}
	if done < 0 {
		done = 0
	}
	if done > total {
		done = total
	}

	filled := int(float64(done) * float64(width) / float64(total))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func init() {
	rootCmd.AddCommand(newDocsCaptureCmd())
}
