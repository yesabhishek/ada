package cli

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yesabhishek/ada/internal/buildinfo"
	"github.com/yesabhishek/ada/internal/gitutil"
	"github.com/yesabhishek/ada/internal/workspace"
)

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

type doctorReport struct {
	Version     buildinfo.Info `json:"version"`
	Overall     string         `json:"overall"`
	SupportedOS bool           `json:"supported_os"`
	Checks      []doctorCheck  `json:"checks"`
}

func newDoctorCommand() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run local environment checks for the Ada alpha.",
		RunE: func(cmd *cobra.Command, args []string) error {
			report := runDoctor(context.Background())
			if jsonOutput {
				return writeJSON(cmd.OutOrStdout(), report)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Ada Doctor\n%s\n\n", buildinfo.HumanString())
			for _, check := range report.Checks {
				fmt.Fprintf(cmd.OutOrStdout(), "- [%s] %s: %s\n", strings.ToUpper(check.Status), check.Name, check.Details)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\nOverall: %s\n", strings.ToUpper(report.Overall))
			if report.Overall == "fail" {
				return dependencyErrorf("doctor found one or more blocking issues")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output doctor report as JSON")
	return cmd
}

func runDoctor(ctx context.Context) doctorReport {
	report := doctorReport{
		Version: buildinfo.Current(),
	}

	supportedOS := runtime.GOOS == "darwin" || runtime.GOOS == "linux"
	report.SupportedOS = supportedOS
	if supportedOS {
		report.Checks = append(report.Checks, doctorCheck{Name: "platform", Status: "ok", Details: fmt.Sprintf("%s/%s is a supported alpha target", runtime.GOOS, runtime.GOARCH)})
	} else {
		report.Checks = append(report.Checks, doctorCheck{Name: "platform", Status: "fail", Details: fmt.Sprintf("%s/%s is not part of the public alpha support matrix", runtime.GOOS, runtime.GOARCH)})
	}

	if gitPath, err := exec.LookPath("git"); err == nil {
		report.Checks = append(report.Checks, doctorCheck{Name: "git", Status: "ok", Details: gitPath})
	} else {
		report.Checks = append(report.Checks, doctorCheck{Name: "git", Status: "fail", Details: "git is required but was not found in PATH"})
	}

	if prettierPath, err := exec.LookPath("prettier"); err == nil {
		report.Checks = append(report.Checks, doctorCheck{Name: "prettier", Status: "ok", Details: prettierPath})
	} else {
		report.Checks = append(report.Checks, doctorCheck{Name: "prettier", Status: "warn", Details: "optional; TypeScript files will not be auto-formatted without prettier"})
	}

	if repo, err := gitutil.Discover(ctx, "."); err == nil {
		statusEntries, _ := repo.StatusEntries(ctx)
		details := repo.Root
		if len(statusEntries) > 0 {
			details += " (dirty working tree)"
			report.Checks = append(report.Checks, doctorCheck{Name: "git-repo", Status: "warn", Details: details})
		} else {
			report.Checks = append(report.Checks, doctorCheck{Name: "git-repo", Status: "ok", Details: details})
		}
	} else {
		report.Checks = append(report.Checks, doctorCheck{Name: "git-repo", Status: "warn", Details: "no git repository detected in the current directory tree"})
	}

	if ws, err := workspace.Find("."); err == nil {
		report.Checks = append(report.Checks, doctorCheck{Name: "ada-workspace", Status: "ok", Details: filepath.Join(ws.Root, ".ada")})
	} else {
		report.Checks = append(report.Checks, doctorCheck{Name: "ada-workspace", Status: "warn", Details: "no Ada workspace found; run `ada start .` in a git repo"})
	}

	report.Overall = "ok"
	for _, check := range report.Checks {
		if check.Status == "fail" {
			report.Overall = "fail"
			return report
		}
		if check.Status == "warn" {
			report.Overall = "warn"
		}
	}
	return report
}
