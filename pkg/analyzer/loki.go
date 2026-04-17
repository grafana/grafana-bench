package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LokiConfig configures the Loki HTTP client.
type LokiConfig struct {
	BaseURL     string
	Username    string
	Password    string
	BearerToken string
	HTTPTimeout time.Duration
	ChunkSize   time.Duration
	MaxRetries  int
}

// LokiClient is a minimal query_range client tailored to the analyzer's
// needs: chunked time windows, bearer-or-basic auth, and structured logging
// of the underlying HTTP traffic.
type LokiClient struct {
	cfg   LokiConfig
	http  *http.Client
	log   *slog.Logger
	rng   *rand.Rand
	chunk time.Duration
	tries int
	sleep func(time.Duration)
}

// NewLokiClient constructs a LokiClient and applies sensible defaults.
func NewLokiClient(cfg LokiConfig, log *slog.Logger) *LokiClient {
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 60 * time.Second
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = time.Hour
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	return &LokiClient{
		cfg:   cfg,
		http:  &http.Client{Timeout: cfg.HTTPTimeout},
		log:   log,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
		chunk: cfg.ChunkSize,
		tries: cfg.MaxRetries,
		sleep: time.Sleep,
	}
}

type lokiQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string            `json:"resultType"`
		Result     []lokiStreamEntry `json:"result"`
	} `json:"data"`
}

type lokiStreamEntry struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"` // [ns_timestamp_string, line_string]
}

// QueryTestRuns executes a LogQL query_range for the given stream selector
// over [start, end], chunking the request into ChunkSize windows so that
// Loki's per-query limit doesn't silently truncate results.
func (c *LokiClient) QueryTestRuns(ctx context.Context, selector string, start, end time.Time) ([]TestRunRecord, error) {
	if c.cfg.BaseURL == "" {
		return nil, fmt.Errorf("loki base url is required")
	}
	if !end.After(start) {
		return nil, fmt.Errorf("end must be after start")
	}

	var all []TestRunRecord
	cursor := start
	for cursor.Before(end) {
		chunkEnd := cursor.Add(c.chunk)
		if chunkEnd.After(end) {
			chunkEnd = end
		}
		batch, err := c.queryChunk(ctx, selector, cursor, chunkEnd)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		cursor = chunkEnd
	}

	sort.Slice(all, func(i, j int) bool { return all[i].Time.Before(all[j].Time) })
	return all, nil
}

func (c *LokiClient) queryChunk(ctx context.Context, selector string, start, end time.Time) ([]TestRunRecord, error) {
	u, err := url.Parse(strings.TrimRight(c.cfg.BaseURL, "/") + "/loki/api/v1/query_range")
	if err != nil {
		return nil, fmt.Errorf("parsing loki url: %w", err)
	}
	q := u.Query()
	q.Set("query", selector)
	q.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	q.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	q.Set("direction", "forward")
	q.Set("limit", "5000")
	u.RawQuery = q.Encode()

	var body []byte
	for attempt := 0; attempt <= c.tries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("building loki request: %w", err)
		}
		c.applyAuth(req)

		resp, err := c.http.Do(req)
		if err != nil {
			if attempt < c.tries {
				c.backoff(attempt)
				continue
			}
			return nil, fmt.Errorf("loki request: %w", err)
		}
		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading loki response: %w", err)
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt < c.tries {
				c.log.Debug("loki retryable status, backing off",
					"status", resp.StatusCode, "attempt", attempt+1)
				c.backoff(attempt)
				continue
			}
			return nil, fmt.Errorf("loki returned %d after %d attempts: %s", resp.StatusCode, attempt+1, truncate(string(body), 500))
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("loki returned %d: %s", resp.StatusCode, truncate(string(body), 500))
		}
		break
	}

	var decoded lokiQueryResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decoding loki response: %w", err)
	}
	if decoded.Status != "success" {
		return nil, fmt.Errorf("loki query status %q", decoded.Status)
	}

	var out []TestRunRecord
	for _, stream := range decoded.Data.Result {
		for _, v := range stream.Values {
			if len(v) != 2 {
				continue
			}
			rec, err := decodeRecord([]byte(v[1]))
			if err != nil {
				c.log.Debug("skipping undecodable loki line", "err", err)
				continue
			}
			if rec.Time.IsZero() {
				if ns, err := strconv.ParseInt(v[0], 10, 64); err == nil {
					rec.Time = time.Unix(0, ns)
				}
			}
			out = append(out, rec)
		}
	}
	return out, nil
}

func (c *LokiClient) applyAuth(req *http.Request) {
	if c.cfg.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.BearerToken)
		return
	}
	if c.cfg.Username != "" || c.cfg.Password != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}
}

func (c *LokiClient) backoff(attempt int) {
	base := time.Duration(1<<attempt) * 500 * time.Millisecond
	jitter := time.Duration(c.rng.Int63n(int64(250 * time.Millisecond)))
	c.sleep(base + jitter)
}
