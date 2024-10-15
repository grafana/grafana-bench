package recorder

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

const capture = `
127.0.0.1:40288: GET https://instance.grafana.net/api/frontend/settings HTTP/2.0
     << HTTP/2.0 200 OK 10.56k
127.0.0.1:40298: POST https://instance.grafana.net/login HTTP/2.0
     << HTTP/2.0 200 OK 41b
127.0.0.1:40302: POST https://instance.grafana.net/api/dashboards/db HTTP/2.0
     << HTTP/2.0 200 OK 175b
127.0.0.1:40318: DELETE https://instance.grafana.net/api/dashboards/uid/jQzZSWJqLhLeoUv HTTP/2.0
     << HTTP/2.0 200 OK 123b
`

func TestParseRecording(t *testing.T) {
	testCases := []struct {
		testCase  string
		expectErr error
		expect    Recording
	}{
		{
			testCase: "valid recording",
			expect: Recording{
				Requests: []Request{
					{
						Host:   "instance.grafana.net",
						Path:   "/api/frontend/settings",
						Method: "GET",
						Status: "200",
					},
					{
						Host:   "instance.grafana.net",
						Path:   "/login",
						Method: "POST",
						Status: "200",
					},
					{
						Host:   "instance.grafana.net",
						Path:   "/api/dashboards/db",
						Method: "POST",
						Status: "200",

					},
					{
						Host:   "instance.grafana.net",
						Path:   "/api/dashboards/uid/jQzZSWJqLhLeoUv",
						Method: "DELETE",
						Status: "200",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCase, func(t *testing.T) {
			recording, err := ParseRecording([]byte(capture))
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected error %v got %v", tc.expectErr, err)
			}

			if tc.expectErr != nil {
				return
			}

			if !reflect.DeepEqual(recording, tc.expect) {
				t.Fatalf("expected %v got %v", tc.expect, recording)
			}
		})
	}
}

type flow struct {
	path   string
	method string
	status int
}

func TestRecordingProxy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testCase  string
		flows     []flow
		recording Recording
	}{
		{
			testCase: "valid recording",
			flows: []flow{
				{
					path:   "/api/frontend/settings",
					method: "GET",
					status: 200,
				},
				{
					path:   "/login",
					method: "POST",
					status: 200,
				},
				{
					path:   "/api/dashboards/db",
					method: "POST",
					status: 200,
				},
			},
			recording: Recording{
				Requests: []Request{
					{
						Path:   "/api/frontend/settings",
						Method: "GET",
						Status: "200",
					},
					{
						Path:   "/login",
						Method: "POST",
						Status: "200",
					},
					{
						Path:   "/api/dashboards/db",
						Method: "POST",
						Status: "200",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testCase, func(t *testing.T) {
			// start an http server that returns 200 for all requests
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer target.Close()

			recorder, err := NewProxyRecorder(ProxyOptions{
				Scheme: "http",
				Target: target.URL,
				Port: 8091,
				Verbose: true,
			})
			if err != nil {
				t.Fatalf("starting recorder: %v", err)
			}

			proxyUrl, err := url.Parse(recorder.ProxyURL())

			client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}

			for _, flow := range tc.flows {
				request, _ := http.NewRequest(flow.method, target.URL+flow.path, nil)

				resp, err := client.Do(request)
				if err != nil {
					t.Fatal("cannot send reque st", err)
				}

				if resp.StatusCode != flow.status {
					t.Fatalf("expected status %v got %v", flow.status, resp.StatusCode)
				}
			}

			recording, err := recorder.GetRecording()
			if err != nil {
				t.Fatalf("getting recording: %v", err)
			}


			for i, rec := range tc.recording.Requests {
				// host can't be compared because depends on a random port
				if recording.Requests[i].Method != rec.Method ||recording.Requests[i].Status != rec.Status {
					t.Fatalf("expected %v got %v", tc.recording, recording)
				}
			}
		})
	}
}