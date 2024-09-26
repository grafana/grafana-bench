package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/slack-go/slack"
)

func TestSlackMarkdownFormatter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title       string
		dashboard   string
		suite       executor.TestSuite
		testRuns    []executor.TestRun
		expected    string
		expectedErr error
	}{
		{
			title: "simple test suite",
			suite: executor.TestSuite{},
			testRuns: []executor.TestRun{
				{TestFile: "test.js", Status: executor.TestFailed},
				{TestFile: "another_test.js", Status: executor.TestFailed},
			},
			expected: "*Suite Run:* 123\n" +
				"- test.js failed\n" +
				"- another_test.js failed\n",
		},
		{
			title: "test suite dashboard tes",
			dashboard: "http://url.to/dashboard",
			suite: executor.TestSuite{},
			testRuns: []executor.TestRun{
				{TestFile: "test.js", Status: executor.TestFailed},
				{TestFile: "another_test.js", Status: executor.TestFailed},
			},
			expected: "*Suite Run:* <http://url.to/dashboard?var-SuiteRun=123|123>\n" +
				"- test.js failed\n" +
				"- another_test.js failed\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			message, err := SlackMarkdownFormatter(tc.dashboard, "123", tc.suite, tc.testRuns)

			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v got %v", tc.expectedErr, err)
			}

			if tc.expectedErr != nil {
				return
			}

			if message != tc.expected {
				t.Fatalf("expected\n%s\ngot\n%s", tc.expected, message)
			}
		})
	}
}

// mockServer is a mock slack server that can be used for testing
// chat.post endpoint registers the messages sent to each channel
// conversations.list returns the list of channels in the server
//
// This mock is inspired in https://github.com/slack-go/slack/blob/master/slacktest
type mockServer struct {
	mutext   *sync.Mutex
	channels map[string]string
	messages map[string][]string
}

func newMockServer(channels map[string]string) *mockServer {
	return &mockServer{
		mutext:   &sync.Mutex{},
		channels: channels,
		messages: map[string][]string{},
	}
}

func (m *mockServer) Messages(channel string) []string {
	m.mutext.Lock()
	defer m.mutext.Unlock()

	messages := []string{}
	messages = append(messages, m.messages[channel]...)
	return messages
}

// HandleChatPostMessage is the handler for the chat.post endpoint of the mock slack
// server. It takes the channel, channel_id, and text form values and
// stores them in the messages map. It then responds with a JSON object
// that contains the channel, ts, and text.
func (m *mockServer) HandleChatPostMessage(w http.ResponseWriter, r *http.Request) {
	m.mutext.Lock()
	defer m.mutext.Unlock()

	channel := r.FormValue("channel")
	text := r.FormValue("text")
	m.messages[channel] = append(m.messages[channel], text)

	resp := &struct {
		Ok      bool   `json:"ok"`
		Channel string `json:"channel"`
		Ts      string `json:"ts"`
		Text    string `json:"text"`
	}{
		Ok:      true,
		Channel: channel,
		Ts:      fmt.Sprintf("%d", time.Now().Unix()),
		Text:    text,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (m *mockServer) HandleConversationsList(w http.ResponseWriter, r *http.Request) {
	resp := struct {
		Ok       bool            `json:"ok"`
		Channels []slack.Channel `json:"channels"`
	}{}

	resp.Ok = true
	for ch, id := range m.channels {
		resp.Channels = append(
			resp.Channels,
			slack.Channel{
				GroupConversation: slack.GroupConversation{
					Name: ch,
					Conversation: slack.Conversation{
						ID: id,
					},
				},
			},
		)
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// TODO: test errors from slack server
func TestNotify(t *testing.T) {
	testCases := []struct {
		title        string
		recipient    string
		testRuns     []executor.TestRun
		expectedMsgs map[string][]string
		expectedErr  error
	}{
		{
			title:     "notify code owner",
			recipient: "codeowner-team",
			testRuns: []executor.TestRun{
				{TestFolder: "test-suite", TestFile: "failed.js", Status: executor.TestFailed},
			},
			expectedMsgs: map[string][]string{"CHANNEL_ID": {"failed.js"}},
			expectedErr:  nil,
		},
		{
			title:        "nothing to notify",
			recipient:    "codeowner-team",
			testRuns:     []executor.TestRun{},
			expectedMsgs: map[string][]string{},
			expectedErr:  nil,
		},
		{
			title:        "recipient's channel not found in mapping",
			recipient:    "another-codeowner-team",
			testRuns:     []executor.TestRun{},
			expectedMsgs: map[string][]string{},
			expectedErr:  ErrNoMappingForCodeowner,
		},
		{
			title:        "recipient's channel does not exist",
			recipient:    "codeowner-with-missing-channel",
			testRuns:     []executor.TestRun{},
			expectedMsgs: map[string][]string{},
			expectedErr:  ErrChannelDoesNotExist,
		},
	}

	channels := map[string]string{
		"codeowner-channel": "CHANNEL_ID",
		"another-channel":   "ANOTHER_CHANNEL_ID",
	}

	mapping := CodeownersMapping{
		"codeowner-team":                 "codeowner-channel",
		"codeowner-with-missing-channel": "missing-channel",
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			mock := newMockServer(channels)

			handler := http.NewServeMux()
			handler.HandleFunc("/chat.postMessage", mock.HandleChatPostMessage)
			handler.HandleFunc("/conversations.list", mock.HandleConversationsList)
			srv := httptest.NewServer(handler)

			notifier, _ := NewSlackNotifier(SlackNotifierOptions{
				Client:  slack.New("FAKE_TOKEN", slack.OptionAPIURL(srv.URL+"/")),
				Mapping: mapping,
			},
			)

			err := notifier.Notify(context.Background(), tc.recipient, "123", tc.testRuns)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v got %v", tc.expectedErr, err)
			}

			for ch, expected := range tc.expectedMsgs {
				messages := mock.Messages(ch)
				if len(messages) != len(expected) {
					t.Fatalf("expected %d messages in channel %s got %d", len(expected), ch, len(mock.messages[ch]))
				}
			}
		})
	}
}
