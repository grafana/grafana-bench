package validate

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-bench/pkg/config"
	"github.com/grafana/grafana-bench/pkg/notifier"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	validateSlackNotifierPermissions bool
)

// NewCmd returns a new bench version command
func NewCmd(log *slog.Logger) *cobra.Command {
	benchConfig := &config.BenchConfig{}

	cmd := cobra.Command{
		Use:     "validate",
		Short:   "validate --<validation>",
		Long:    "validate provides validations for the current bench configuration. currently only codeowners is supported",
		Example: "bench validate --slack-notifier-permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if validateSlackNotifierPermissions {
				err := ValidateSlackNotiferPermissions(benchConfig)
				if err != nil {
					log.Error(err.Error())
				}
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(
		&validateSlackNotifierPermissions,
		"slack-notifier-permissions",
		false,
		"validate slack notifier permissions based on the current codeowner-mapping file",
	)

	config.AddSlackFlags(fs, &benchConfig.Slack)

	return &cmd
}

func ValidateSlackNotiferPermissions(config *config.BenchConfig) error {
	if config.Slack.Token == "" {
		return fmt.Errorf("no slack token provided")
	}

	codeownersMap := config.Slack.CodeownersMap
	if !filepath.IsAbs(codeownersMap) {
		codeownersMap = filepath.Join(config.TestSuite.BaseDir, codeownersMap)
	}

	// NewSlackNotifer returns a notifer interface.
	n, err := notifier.NewSlackNotifier(notifier.SlackNotifierOptions{
		Token:        config.Slack.Token,
		MappingFile:  codeownersMap,
		DashboardURL: config.SuiteRun.DashboardURL,
	})
	if err != nil {
		return fmt.Errorf("creating slack notifier: %w", err)
	}

	// Cast to slackNotifier so we can call check permissions
	slackNotifier := n.(*notifier.SlackNotifier)
	channelStatuses := slackNotifier.CheckPermissions()

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

		table.Append([]string{status.ID, status.Name, s, e})
	}

	table.Render()

	if anyError {
		return fmt.Errorf("Slack bot does not have permissions for one or more channels")
	}

	return nil
}
