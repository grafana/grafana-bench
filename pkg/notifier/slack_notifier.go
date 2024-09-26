package notifier

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/slack-go/slack"
)

var (
	ErrChannelDoesNotExist = errors.New("channel does not exist")
	ErrFormattingMessage   = errors.New("formatting message")
	ErrGettingChannels     = errors.New("getting channels")
	ErrPostingMessage      = errors.New("posting message")
)

const markdownTemplate = `
{{- define  "testSuiteHeader" -}}
{{- if .DashboardURL -}}
*Suite Run:* <{{ .DashboardURL }}?var-SuiteRun={{ .TestSuiteRunId }}|{{ .TestSuiteRunId }}>
{{- else -}}
*Suite Run:* {{ .TestSuiteRunId }}
{{- end -}}
{{- end -}}
{{ template "testSuiteHeader" . }}
{{- range .TestRuns }}
- {{ .TestFile }} {{ .Status }}
{{- end }}
`

type data struct {
	TestSuiteRunId string
	TestRuns       []executor.TestRun
	TestSuite      executor.TestSuite
	DashboardURL   string
}

func SlackMarkdownFormatter(
	dashboard string,
	suiteRunId string,
	suite executor.TestSuite,
	testRuns []executor.TestRun,
) (string, error) {
	formattingTemplate, err := template.New("notification").Parse(markdownTemplate)
	if err != nil {
		return "", fmt.Errorf("%w %w", ErrFormattingMessage, err)
	}

	buffer := new(strings.Builder)
	err = formattingTemplate.Execute(
		buffer, data{
			TestSuiteRunId: suiteRunId,
			TestRuns:       testRuns,
			TestSuite:      suite,
			DashboardURL:   dashboard,
		},
	)
	if err != nil {
		return "", fmt.Errorf("%w %w", ErrFormattingMessage, err)
	}

	return buffer.String(), nil
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
	channel, err := s.mapping.GetChannel(recipient)
	if err != nil {
		return err
	}

	channelID, err := s.getChannelID(ctx, channel)
	if err != nil {
		return err
	}

	if channelID == "" {
		return fmt.Errorf("%w %q", ErrChannelDoesNotExist, channel)
	}

	message, err := SlackMarkdownFormatter(s.dashboardURL, suiteRunId, executor.TestSuite{}, testRuns)
	if err != nil {
		return err
	}

	_, _, err = s.client.PostMessage(channelID, slack.MsgOptionText(message, false))

	if err != nil {
		return fmt.Errorf("%w: %w", ErrPostingMessage, err)
	}

	return nil
}

// getChannelID returns the ID of a Slack channel with the given name.
//
// The ID is cached after the first lookup, so subsequent calls with the same
// channel name are fast. If the channel is not found, an empty string is returned.
func (s *slackNotifier) getChannelID(ctx context.Context, channel string) (string, error) {
	channelID, found := s.channels[channel]

	if channelID != "" {
		return channelID, nil
	}

	// we already looked and didn't find it channelID, don't try again
	if found {
		return "", nil
	}

	cursor := ""
	for {
		channels, nextCursor, err := s.client.GetConversationsContext(
			ctx,
			&slack.GetConversationsParameters{
				Cursor:          cursor,
				Limit:           100,
				ExcludeArchived: true,
				Types:           []string{"public_channel", "private_channel"},
			},
		)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrGettingChannels, err)
		}

		for _, c := range channels {
			if c.Name == channel {
				s.channels[channel] = c.ID
				return c.ID, nil
			}
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	// we did find the channel, save for later
	s.channels[channel] = ""

	return "", nil
}
