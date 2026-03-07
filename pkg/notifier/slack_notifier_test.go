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

// mockServer is a mock slack server that can be used for testing
// chat.post endpoint registers the messages sent to each channel
// conversations.list returns the list of channels in the server
//
// This mock is inspired in https://github.com/slack-go/slack/blob/master/slacktest
type mockServer struct {
	mutext   *sync.Mutex
	channels map[string]string
	messages map[string][]string
	apiErr   string // if non-empty, chat.postMessage returns this Slack API error
}

func newMockServerWithError(apiErr string) *mockServer {
	return &mockServer{
		mutext:   &sync.Mutex{},
		messages: map[string][]string{},
		apiErr:   apiErr,
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
// If the server was configured with an apiErr, it returns an error response instead.
func (m *mockServer) HandleChatPostMessage(w http.ResponseWriter, r *http.Request) {
	m.mutext.Lock()
	defer m.mutext.Unlock()

	if m.apiErr != "" {
		resp := &struct {
			Ok    bool   `json:"ok"`
			Error string `json:"error"`
		}{
			Ok:    false,
			Error: m.apiErr,
		}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

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

func TestNotify(t *testing.T) {
	testCases := []struct {
		title        string
		recipient    string
		testRuns     []executor.TestRunSummary
		serverErr    string // Slack API error code to simulate (e.g. "not_in_channel")
		expectedMsgs map[string][]string
		expectedErr  error
	}{
		{
			title:     "notify code owner",
			recipient: "codeowner-team",
			testRuns: []executor.TestRunSummary{
				{TestFolder: "test-suite", TestFile: "failed.js", Status: executor.TestFailed},
			},
			expectedMsgs: map[string][]string{"CHANNEL_ID": {"failed.js"}},
			expectedErr:  nil,
		},
		{
			title:        "nothing to notify",
			recipient:    "codeowner-team",
			testRuns:     []executor.TestRunSummary{},
			expectedMsgs: map[string][]string{},
			expectedErr:  nil,
		},
		{
			title:        "recipient's channel not found in mapping",
			recipient:    "another-codeowner-team",
			testRuns:     []executor.TestRunSummary{},
			expectedMsgs: map[string][]string{},
			expectedErr:  ErrNoMappingForCodeowner,
		},
		{
			title:     "bot not in channel returns posting error",
			recipient: "codeowner-team",
			testRuns: []executor.TestRunSummary{
				{TestFolder: "test-suite", TestFile: "failed.js", Status: executor.TestFailed},
			},
			serverErr:    "not_in_channel",
			expectedMsgs: map[string][]string{},
			expectedErr:  ErrPostingMessage,
		},
		{
			title:     "channel not found returns posting error",
			recipient: "codeowner-team",
			testRuns: []executor.TestRunSummary{
				{TestFolder: "test-suite", TestFile: "failed.js", Status: executor.TestFailed},
			},
			serverErr:    "channel_not_found",
			expectedMsgs: map[string][]string{},
			expectedErr:  ErrPostingMessage,
		},
	}

	mapping := CodeownersMapping{
		"codeowner-team": "CHANNEL_ID",
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			mock := newMockServerWithError(tc.serverErr)

			handler := http.NewServeMux()
			handler.HandleFunc("/chat.postMessage", mock.HandleChatPostMessage)
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
