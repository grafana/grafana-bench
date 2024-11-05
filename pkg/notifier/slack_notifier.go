package notifier

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-bench/pkg/dashboard"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/slack-go/slack"
)

var (
	ErrChannelDoesNotExist = errors.New("channel does not exist")
	ErrFormattingMessage   = errors.New("formatting message")
	ErrGettingChannels     = errors.New("getting channels")
	ErrPostingMessage      = errors.New("posting message")
)


type data struct {
	TestSuiteRunId string
	TestRuns       []executor.TestRun
	TestSuite      executor.TestSuite
	DashboardURL   string
}

func FormatTestResults(
	dashboardURL string,
	suiteRunId string,
	suite executor.TestSuite,
	testRuns []executor.TestRun,
) ([]slack.Block, error) {
	blocks := []slack.Block{}

	// Header Section
	testRunHeader := suiteRunId

	if dashboardURL != "" {
		dashboardLink, err := dashboard.RenderDashboardURL(dashboardURL, suiteRunId)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFormattingMessage, err)
		}
		testRunHeader = fmt.Sprintf("<%s|%s>", dashboardLink, suiteRunId)
	}
	headerText := slack.NewTextBlockObject(
		"mrkdwn", 
		fmt.Sprintf("*Suite Run:* %s", testRunHeader),
		false,
		false,
	)
	blocks =  append(blocks, slack.NewSectionBlock(headerText, nil, nil))

	// creates a section for each test run with two fields to emulate a table
	for _, testRun := range testRuns {
		testRunFields := []*slack.TextBlockObject{
			slack.NewTextBlockObject(
				"mrkdwn",
				fmt.Sprintf("%s", filepath.Join(testRun.TestFolder, testRun.TestFile)),
				false,
				false,
			),
			slack.NewTextBlockObject(
				"mrkdwn",
				fmt.Sprintf("*%s*", testRun.Status),
				false,
				false,
			),
		}
		testRunSection := slack.NewSectionBlock(
			nil,
			testRunFields,
			nil,
		)

		blocks = append(blocks, testRunSection)
	}

	return blocks, nil
}

type slackNotifier struct {
	client       *slack.Client
	mapping      CodeownersMapping
	channels     map[string]string
	dashboardURL string
}

// Options for creating a new SlackNotifier
type SlackNotifierOptions struct {
	// Slack token. If not specified, the environment variable SLACK_TOKEN must be set.
	Token string
	// MappingFile is the path to the codeowners mapping file. Can be path or an URL
	MappingFile string
	// Mapping maps the codeowners to slack channel
	Mapping CodeownersMapping
	// Client is used for testing purposes
	Client *slack.Client
	// URL to dashboard
	DashboardURL string
}

// NewSlackNotifier returns a Notifier that sends notifications to a Slack channel.
// If the Client is not provided, a new client will be created using the Slack token.
// The Slack token can be provided as an option, or it can be set as the
// environment variable SLACK_TOKEN.
// The slack token requires chat.write, channel.read, groups.read scopes
func NewSlackNotifier(options SlackNotifierOptions) (Notifier, error) {
	client := options.Client
	if client == nil {
		token := options.Token
		if token == "" {
			token = os.Getenv("SLACK_TOKEN")
		}
		client = slack.New(token)
	}

	mapping := options.Mapping
	if mapping == nil {
		var err error
		mapping, err = NewCodeownersMapping(options.MappingFile)
		if err != nil {
			return nil, err
		}
	}
	return &slackNotifier{
		client:       client,
		mapping:      mapping,
		channels:     make(map[string]string),
		dashboardURL: options.DashboardURL,
	}, nil
}

func (s *slackNotifier) Notify(
	ctx context.Context,
	recipient string,
	suiteRunId string,
	testRuns []executor.TestRun,
) error {
	channelID, err := s.mapping.GetChannel(recipient)
	if err != nil {
		return err
	}

	blocks, err := FormatTestResults(s.dashboardURL, suiteRunId, executor.TestSuite{}, testRuns)
	if err != nil {
		return err
	}

	_, _, err = s.client.PostMessage(channelID, slack.MsgOptionBlocks(blocks...))

	if err != nil {
		return fmt.Errorf("%w: %w", ErrPostingMessage, err)
	}

	return nil
}

