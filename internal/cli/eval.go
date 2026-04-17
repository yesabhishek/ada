package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yesabhishek/ada/internal/evals"
	"github.com/yesabhishek/ada/internal/language"
	golanguage "github.com/yesabhishek/ada/internal/language/golang"
	tslanguage "github.com/yesabhishek/ada/internal/language/typescript"
)

func newEvalCommand() *cobra.Command {
	var (
		format         string
		languageFilter string
		scenarioNames  []string
		jsonOutput     bool
	)
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Run built-in Git vs Ada evaluation scenarios.",
		RunE: func(cmd *cobra.Command, args []string) error {
			registry := language.NewRegistry(golanguage.New(), tslanguage.New())
			scenarios := evals.SelectScenarios(evals.DefaultScenarios(), scenarioNames, strings.TrimSpace(languageFilter))
			if len(scenarios) == 0 {
				return usageErrorf("no eval scenarios matched the provided filters")
			}
			report, err := evals.Run(context.Background(), registry, scenarios)
			if err != nil {
				return runtimeError("run eval scenarios", err)
			}
			if jsonOutput {
				format = "json"
			}
			switch format {
			case "text":
				fmt.Fprint(cmd.OutOrStdout(), evals.RenderText(report))
			case "markdown", "md":
				fmt.Fprint(cmd.OutOrStdout(), evals.RenderMarkdown(report))
			case "json":
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			default:
				return fmt.Errorf("unsupported format %q", format)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "output format: text, markdown, or json")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output eval report as JSON")
	cmd.Flags().StringVar(&languageFilter, "language", "", "filter scenarios by language")
	cmd.Flags().StringSliceVar(&scenarioNames, "scenario", nil, "run only the named scenario(s)")
	return cmd
}
