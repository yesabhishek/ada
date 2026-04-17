package evals

import (
	"context"
	"os/exec"
	"testing"

	"github.com/yesabhishek/ada/internal/language"
	golanguage "github.com/yesabhishek/ada/internal/language/golang"
	tslanguage "github.com/yesabhishek/ada/internal/language/typescript"
)

func TestDefaultScenariosProduceExpectedVerdicts(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	registry := language.NewRegistry(golanguage.New(), tslanguage.New())
	report, err := Run(context.Background(), registry, DefaultScenarios())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := report.Summary.Total, len(DefaultScenarios()); got != want {
		t.Fatalf("summary total = %d, want %d", got, want)
	}
	expected := make(map[string]string)
	for _, scenario := range DefaultScenarios() {
		expected[scenario.Name] = scenario.ExpectVerdict
	}
	for _, result := range report.Results {
		if want := expected[result.Name]; result.Verdict != want {
			t.Fatalf("scenario %s verdict = %s, want %s", result.Name, result.Verdict, want)
		}
	}
	if report.Summary.AdaAdvantage == 0 {
		t.Fatalf("expected at least one ada advantage scenario")
	}
	if report.Summary.GitAdvantage == 0 {
		t.Fatalf("expected at least one git advantage scenario")
	}
}
