package validate

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-bench/pkg/config"
	"github.com/grafana/grafana-bench/pkg/validate"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	validateSlackNotifierPermissions bool
)

const example = `
Validate slack configuration
  bench validate --check-slack-permissions

  Requires the --slack-token and the --codeowners-mapping flags

`

// NewCmd returns a new bench version command
func NewCmd(log *slog.Logger) *cobra.Command {
	benchConfig := &config.BenchConfig{}

	cmd := cobra.Command{
		Use:     "validate",
		Short:   "validate",
		Long:    "validate provides validations for the bench configuration.",
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			if validateSlackNotifierPermissions {
				return ValidateSlackNotiferPermissions(benchConfig)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(
		&validateSlackNotifierPermissions,
		"check-slack-permissions",
		false,
		"validate slack notifier permissions based on the current codeowner-mapping file",
	)

	config.AddSlackToken(fs, &benchConfig.Slack)
	config.AddSlackCodeownersMapFlag(fs, &benchConfig.Slack)

	return &cmd
}

func ValidateSlackNotiferPermissions(config *config.BenchConfig) error {
	if config.Slack.Token == "" {
		return fmt.Errorf("no slack token provided")
	}

	codeownersMap := config.Slack.CodeownersMap
	if codeownersMap == "" {
		return fmt.Errorf("no codeowners mapping provided")
	}
	if !filepath.IsAbs(codeownersMap) {
		codeownersMap = filepath.Join(config.TestSuite.BaseDir, codeownersMap)
	}

	channelStatuses, err := validate.CheckPermissions(codeownersMap, config.Slack.Token)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Channel ID", "Channel Name", "Status", "Error"})

	anyError := false
	for _, status := range channelStatuses {
		s := "ok!"
		e := ""
		if status.Err != nil {
			anyError = true
			s = "error"
			e = status.Err.Error()
		}

		if err = table.Append([]string{status.ID, status.Name, s, e}); err != nil {
			return err
		}
	}

	if err = table.Render(); err != nil {
		return err
	}

	if anyError {
		return fmt.Errorf("slack bot does not have permissions for one or more channels")
	}

	return nil
}
