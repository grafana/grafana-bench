package test

import (
	"log/slog"
	"os"
	"testing"
)

var sampleCloudRunOutput = `
          /\      |‾‾| /‾‾/   /‾‾/
     /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
   /          \   |  |\  \ |  (‾)  |
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: /home/k6/work/test/suite/tests/dashboards/dashboard_update.js
     output: cloud (https://ops.grafana.net/a/k6-app/runs/1938289), json (/tmp/dashboard_update.json)

  scenarios: (100.00%) 1 scenario, 7 max VUs, 1h0m30s max duration (incl. graceful stop):
           * dashboardUpdate: 100 iterations shared among 7 VUs (maxDuration: 1h0m0s, gracefulStop: 30s)
`

func TestParseK6CloudIdentifiersFromCLIOutput(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	testCases := []struct {
		title       string
		input       []byte
		expectedID  string
		expectedURL string
		wantErr     bool
	}{
		{
			title:       "parse url and id",
			input:       []byte(sampleCloudRunOutput),
			expectedID:  "1938289",
			expectedURL: "https://ops.grafana.net/a/k6-app/runs/1938289",
			wantErr:     false,
		},
		{
			title:       "none url found",
			input:       []byte(""),
			expectedID:  "",
			expectedURL: "",
			wantErr:     true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.title, func(t *testing.T) {
			id, url, err := parseK6CloudIdentifiersFromCLIOutput(log, testCase.input)

			if (err != nil) != testCase.wantErr {
				t.Errorf("Expected error: %v, but got error: %v", testCase.wantErr, err)
			}

			if id != testCase.expectedID {
				t.Errorf("Expected: %s, but got: %s", testCase.expectedID, id)
			}

			if url != testCase.expectedURL {
				t.Errorf("Expected: %s, but got: %s", testCase.expectedID, id)
			}
		})
	}
}

var iterationOutputFull = `
          /\      |‾‾| /‾‾/   /‾‾/
     /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
   /          \   |  |\  \ |  (‾)  |
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: /home/k6/work/test/suite/tests/dashboards/dashboard_update.js
     output: cloud (https://ops.grafana.net/a/k6-app/runs/1938289), json (/tmp/dashboard_update.json)

  scenarios: (100.00%) 1 scenario, 7 max VUs, 1h0m30s max duration (incl. graceful stop):
           * dashboardUpdate: 100 iterations shared among 7 VUs (maxDuration: 1h0m0s, gracefulStop: 30s)

Run               [ 100% ] setup()
dashboardUpdate   [   0% ]

running (0h00m36.0s), 0/7 VUs, 100 complete and 0 interrupted iterations
dashboardUpdate ✓ [ 100% ] 7 VUs  0h00m03.7s/1h0m0s  100/100 shared iters

     ✗ update dashboard status is 200
      ↳  67% — ✓ 67 / ✗ 33
     ✗ get correct dashboard
      ↳  67% — ✓ 67 / ✗ 33

     █ setup

       ✓ successfully logged in
       ✓ create dashboard status is 200
       ✓ get correct dashboard

     █ teardown

       ✗ delete dashboard status is 200
        ↳  0% — ✓ 0 / ✗ 1

     checks.........................: 83.33% ✓ 335      ✗ 67
     data_received..................: 271 kB 7.5 kB/s
     data_sent......................: 1.4 MB 38 kB/s
     http_req_blocked...............: avg=8.22ms   min=2.45µs   med=8.79µs   max=241.28ms p(90)=17.05µs  p(95)=65.21µs
     http_req_connecting............: avg=2.9ms    min=0s       med=0s       max=83.12ms  p(90)=0s       p(95)=0s
     http_req_duration..............: avg=257.24ms min=140.8ms  med=267.88ms max=561.79ms p(90)=309.04ms p(95)=337.29ms
       { expected_response:true }...: avg=268.01ms min=169.22ms med=275.5ms  max=561.79ms p(90)=317.24ms p(95)=353.63ms
     http_req_failed................: 16.83% ✓ 34       ✗ 168
     http_req_receiving.............: avg=7.93ms   min=139.95µs med=6.19ms   max=27.19ms  p(90)=17.84ms  p(95)=19.64ms
     http_req_sending...............: avg=1.22ms   min=296.16µs med=934.81µs max=7.14ms   p(90)=2.25ms   p(95)=3.06ms
     http_req_tls_handshaking.......: avg=5.04ms   min=0s       med=0s       max=133.15ms p(90)=0s       p(95)=0s
     http_req_waiting...............: avg=248.08ms min=133.05ms med=259.37ms max=534.42ms p(90)=299.96ms p(95)=330.56ms
     http_reqs......................: 202    5.587236/s
     iteration_duration.............: avg=549.7ms  min=155.53ms med=223.41ms max=31.76s   p(90)=310.04ms p(95)=409.85ms
     iterations.....................: 100    2.765958/s
     vus............................: 0      min=0      max=7
     vus_max........................: 7      min=7      max=7
`

var iterationOutput1 = `
     iteration_duration.............: avg=549.7ms  min=155.53ms med=223.41ms max=31.76s   p(90)=310.04ms p(95)=409.85ms
     iterations.....................: 1      2.765958/s
     vus............................: 0      min=0      max=7
`

var iterationOutput100 = `
     iteration_duration.............: avg=549.7ms  min=155.53ms med=223.41ms max=31.76s   p(90)=310.04ms p(95)=409.85ms
     iterations.....................: 100    2.765958/s
     vus............................: 0      min=0      max=7
`

var iterationOuputMissing = `
     http_reqs......................: 202    5.587236/s
     iteration_duration.............: avg=549.7ms  min=155.53ms med=223.41ms max=31.76s   p(90)=310.04ms p(95)=409.85ms
     vus............................: 0      min=0      max=7
     vus_max........................: 7      min=7      max=7
`

func TestParseIterationCountFromCLIOutput(t *testing.T) {
	testCases := []struct {
		title    string
		input    []byte
		expected string
		wantErr  bool
	}{
		{
			title:    "parse 1 from small output sample",
			input:    []byte(iterationOutput1),
			expected: "1",
			wantErr:  false,
		},
		{
			title:    "parse 100 from small output sample",
			input:    []byte(iterationOutput100),
			expected: "100",
			wantErr:  false,
		},
		{
			title:    "parse 100 from large output",
			input:    []byte(iterationOutputFull),
			expected: "100",
			wantErr:  false,
		},
		{
			title:    "parse 1 from small sample",
			input:    []byte(iterationOuputMissing),
			expected: "",
			wantErr:  true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.title, func(t *testing.T) {
			actual, err := parseIterationCountFromCLIOutput(testCase.input)

			if (err != nil) != testCase.wantErr {
				t.Errorf("Expected error: %v, but got error: %v", testCase.wantErr, err)
			}

			if actual != testCase.expected {
				t.Errorf("Expected: %s, but got: %s", testCase.expected, actual)
			}
		})
	}
}
