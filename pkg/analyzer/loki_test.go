package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// buildLokiBody encodes a minimal query_range success payload whose values
// cover the given records in UnixNano-ordered pairs.
func buildLokiBody(recs []TestRunRecord) []byte {
	values := make([][]string, 0, len(recs))
	for _, r := range recs {
		line, _ := json.Marshal(r)
		values = append(values, []string{strconv.FormatInt(r.Time.UnixNano(), 10), string(line)})
	}
	resp := lokiQueryResponse{Status: "success"}
	resp.Data.ResultType = "streams"
	resp.Data.Result = []lokiStreamEntry{{
		Stream: map[string]string{"service_name": "grafana-athena-datasource"},
		Values: values,
	}}
	b, _ := json.Marshal(resp)
	return b
}

func TestLokiClientPaginatesByTimeChunks(t *testing.T) {
	start := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		q := r.URL.Query()
		chunkStart, _ := strconv.ParseInt(q.Get("start"), 10, 64)
		rec := TestRunRecord{
			Time:     time.Unix(0, chunkStart).Add(5 * time.Minute),
			Msg:      "testRun",
			Service:  "grafana-athena-datasource",
			TestFile: "permissions.ts",
			Status:   "failed",
			RunID:    fmt.Sprintf("r-%d", chunkStart),
		}
		w.Write(buildLokiBody([]TestRunRecord{rec}))
	}))
	defer srv.Close()

	c := NewLokiClient(LokiConfig{BaseURL: srv.URL, ChunkSize: time.Hour}, testLogger())
	recs, err := c.QueryTestRuns(context.Background(), "{foo=\"bar\"}", start, end)
	if err != nil {
		t.Fatalf("QueryTestRuns: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("expected 3 chunked calls, got %d", got)
	}
	if len(recs) != 3 {
		t.Errorf("expected 3 records (1 per chunk), got %d", len(recs))
	}
	for i := 1; i < len(recs); i++ {
		if recs[i].Time.Before(recs[i-1].Time) {
			t.Errorf("expected records sorted by time, got %v then %v", recs[i-1].Time, recs[i].Time)
		}
	}
}

func TestLokiClientRetriesOn429(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write(buildLokiBody(nil))
	}))
	defer srv.Close()

	c := NewLokiClient(LokiConfig{BaseURL: srv.URL, ChunkSize: time.Hour, MaxRetries: 5}, testLogger())
	c.sleep = func(time.Duration) {}

	start := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
	_, err := c.QueryTestRuns(context.Background(), "{foo=\"bar\"}", start, start.Add(time.Hour))
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("expected 3 calls (2 retries + 1 success), got %d", got)
	}
}

func TestLokiClientBearerAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer xyz" {
			t.Errorf("expected bearer auth, got %q", got)
		}
		w.Write(buildLokiBody(nil))
	}))
	defer srv.Close()

	c := NewLokiClient(LokiConfig{BaseURL: srv.URL, ChunkSize: time.Hour, BearerToken: "xyz"}, testLogger())
	start := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
	if _, err := c.QueryTestRuns(context.Background(), "{foo=\"bar\"}", start, start.Add(time.Hour)); err != nil {
		t.Fatalf("QueryTestRuns: %v", err)
	}
}

func TestLokiClientBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "user" || p != "pass" {
			t.Errorf("expected basic auth user/pass, got ok=%v u=%q p=%q", ok, u, p)
		}
		w.Write(buildLokiBody(nil))
	}))
	defer srv.Close()

	c := NewLokiClient(LokiConfig{BaseURL: srv.URL, ChunkSize: time.Hour, Username: "user", Password: "pass"}, testLogger())
	start := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
	if _, err := c.QueryTestRuns(context.Background(), "{foo=\"bar\"}", start, start.Add(time.Hour)); err != nil {
		t.Fatalf("QueryTestRuns: %v", err)
	}
}

func TestLokiClientResultsSortedAcrossChunks(t *testing.T) {
	// Each chunk returns records ordered oldest-first, but the concatenation
	// must still be globally time-ordered.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		chunkStart, _ := strconv.ParseInt(q.Get("start"), 10, 64)
		base := time.Unix(0, chunkStart)
		recs := []TestRunRecord{
			{Time: base.Add(10 * time.Minute), Msg: "testRun", Service: "s", TestFile: "t.ts", Status: "passed", RunID: "a"},
			{Time: base.Add(2 * time.Minute), Msg: "testRun", Service: "s", TestFile: "t.ts", Status: "passed", RunID: "b"},
		}
		w.Write(buildLokiBody(recs))
	}))
	defer srv.Close()

	c := NewLokiClient(LokiConfig{BaseURL: srv.URL, ChunkSize: time.Hour}, testLogger())
	start := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
	recs, err := c.QueryTestRuns(context.Background(), "{foo=\"bar\"}", start, start.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("QueryTestRuns: %v", err)
	}
	if len(recs) != 4 {
		t.Fatalf("expected 4 records, got %d", len(recs))
	}
	for i := 1; i < len(recs); i++ {
		if recs[i].Time.Before(recs[i-1].Time) {
			t.Errorf("records not sorted: %v then %v", recs[i-1].Time, recs[i].Time)
		}
	}
}

func TestLokiClientPropagates5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := NewLokiClient(LokiConfig{BaseURL: srv.URL, ChunkSize: time.Hour, MaxRetries: 1}, testLogger())
	c.sleep = func(time.Duration) {}
	start := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
	_, err := c.QueryTestRuns(context.Background(), "{foo=\"bar\"}", start, start.Add(time.Hour))
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500-bearing error, got %v", err)
	}
}

