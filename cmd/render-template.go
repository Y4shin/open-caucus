package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Y4shin/conference-tool/internal/locale"
	"github.com/Y4shin/conference-tool/internal/templrender"
	"github.com/spf13/cobra"
)

type renderTemplatePayload struct {
	Context templrender.ContextProfile `json:"context"`
	Input   json.RawMessage            `json:"input"`
}

func decodeRenderTemplatePayload(component string, r io.Reader) (*renderTemplatePayload, any, error) {
	input, err := templrender.NewInput(component)
	if err != nil {
		return nil, nil, err
	}

	var payload renderTemplatePayload
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, nil, fmt.Errorf("decode render payload: %w", err)
	}

	rawInput := payload.Input
	if len(rawInput) == 0 {
		rawInput = []byte("{}")
	}
	if err := json.Unmarshal(rawInput, input); err != nil {
		return nil, nil, fmt.Errorf("decode component input: %w", err)
	}

	return &payload, input, nil
}

var renderTemplateCmd = &cobra.Command{
	Use:          "render-template <component-name>",
	Short:        "Render a legacy templ component to HTML",
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := locale.LoadTranslations(); err != nil {
			return fmt.Errorf("load translations: %w", err)
		}

		componentName := args[0]
		inputPath, _ := cmd.Flags().GetString("input")

		var reader io.Reader = os.Stdin
		if inputPath != "" {
			file, err := os.Open(inputPath)
			if err != nil {
				return fmt.Errorf("open input file: %w", err)
			}
			defer file.Close()
			reader = file
		}

		payload, input, err := decodeRenderTemplatePayload(componentName, reader)
		if err != nil {
			return err
		}
		return templrender.Write(cmd.OutOrStdout(), componentName, payload.Context, input)
	},
}

func init() {
	renderTemplateCmd.Flags().String("input", "", "Path to JSON input file (defaults to stdin)")
	rootCmd.AddCommand(renderTemplateCmd)
}
