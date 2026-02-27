package cmd

import (
	"fmt"
	"slices"

	"github.com/Y4shin/conference-tool/tools/docscapture"
	"github.com/Y4shin/conference-tool/tools/docscapture/scripts"
	"github.com/spf13/cobra"
)

func newDocsCaptureCmd() *cobra.Command {
	var (
		baseURL        string
		outDir         string
		viewportWidth  int
		viewportHeight int
		headed         bool
		slowMoMS       float64
	)

	cmd := &cobra.Command{
		Use:   "docs-capture",
		Short: "Capture screenshots and GIFs for internal documentation",
		Long: `Capture deterministic visual artifacts from a running conference-tool instance.
Capture scripts are selected by glob and executed for one or more theme/language variants.`,
	}

	cmd.PersistentFlags().StringVar(&baseURL, "base-url", "http://127.0.0.1:8080", "Base URL of the running app")
	cmd.PersistentFlags().StringVar(&outDir, "out-dir", "docs/assets/captures", "Output directory for generated artifacts")
	cmd.PersistentFlags().IntVar(&viewportWidth, "viewport-width", 1440, "Browser viewport width in pixels")
	cmd.PersistentFlags().IntVar(&viewportHeight, "viewport-height", 900, "Browser viewport height in pixels")
	cmd.PersistentFlags().BoolVar(&headed, "headed", false, "Run Chromium in headed mode")
	cmd.PersistentFlags().Float64Var(&slowMoMS, "slow-mo-ms", 0, "Slow down each Playwright operation by N milliseconds")

	commonOptions := func() docscapture.CommonOptions {
		return docscapture.CommonOptions{
			BaseURL:        baseURL,
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
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run capture scripts selected by glob",
		Long: `Run one or more capture scripts against a running app instance.
Each selected script is executed for each selected (theme, language) variant.`,
		Example: fmt.Sprintf(
			`  %s docs-capture run --script "example.*" --theme light --language en
  %s docs-capture run --script "*.gif-*" --theme light,dark --language english,german`,
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
			variants := cartesianVariants(themes, langs)
			if len(variants) == 0 {
				return fmt.Errorf("no variants selected")
			}

			for _, script := range selected {
				for _, variant := range variants {
					outPath, err := script.Run(commonOptions(), variant)
					if err != nil {
						return fmt.Errorf(
							"script %q failed for language=%s theme=%s: %w",
							script.Name,
							variant.Language,
							variant.Theme,
							err,
						)
					}
					cmd.Printf(
						"Generated %-32s language=%-2s theme=%-5s -> %s\n",
						script.Name,
						variant.Language,
						variant.Theme,
						outPath,
					)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&scriptGlobs, "script", []string{"*"}, "Script name globs to include (repeatable)")
	cmd.Flags().StringSliceVar(&rawThemes, "theme", []string{"light", "dark"}, "Theme variants: light,dark (aliases: lightmode,darkmode)")
	cmd.Flags().StringSliceVar(&rawLangs, "language", []string{"en", "de"}, "Language variants: en,de (aliases: english,german,deutsch)")
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

func cartesianVariants(themes []docscapture.Theme, languages []docscapture.Language) []docscapture.Variant {
	variants := make([]docscapture.Variant, 0, len(themes)*len(languages))
	for _, language := range languages {
		for _, theme := range themes {
			variants = append(variants, docscapture.Variant{
				Theme:    theme,
				Language: language,
			})
		}
	}
	return variants
}

func init() {
	rootCmd.AddCommand(newDocsCaptureCmd())
}
