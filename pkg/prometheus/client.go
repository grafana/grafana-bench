// Package: prometheus prometheus push client
package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"time"

	"golang.org/x/exp/slog"
	"google.golang.org/protobuf/proto"

	prompb "buf.build/gen/go/prometheus/prometheus/protocolbuffers/go"
	"github.com/golang/snappy"
)

type Options struct {
	User     string
	Password string
	Debug    bool
	Headers  map[string]string
	Timeout  time.Duration
}

type Client struct {
	debug    bool
	client   *http.Client
	Headers  map[string]string
	url      string
	user     string
	password string
}

func New(url string, options Options) *Client {
	return &Client{
		client: &http.Client{
			Timeout: options.Timeout,
		},
		debug:   options.Debug,
		Headers: options.Headers,
		url:     url,
		user:    options.User,
		password: options.Password,
	}
}

// Push timeseries to a remote write url
func (c *Client) Push(ctx context.Context, series []*prompb.TimeSeries) error {
	// Marshal the data into a byte slice using the protobuf library.
	data, err := proto.Marshal(&prompb.WriteRequest{
		Timeseries: series,
	})
	if err != nil {
		return fmt.Errorf("encoding series as protobuf write request failed: %w", err)
	}

	// Encode the content into snappy encoding.
	compressed := snappy.Encode(nil, data)

	// Create an HTTP request from the body content and set necessary parameters.
	req, err := http.NewRequest("POST", c.url, bytes.NewReader(compressed))
	if err != nil {
		return err
	}

	// Set custom HTTP headers
	for h, v := range c.Headers {
		req.Header[h] = []string{v}
	}

	// Set the required HTTP header content.
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	req.Header.Set("User-Agent", "grafana-bench")
	if c.user != "" {
		req.SetBasicAuth(c.user, c.password)
	}

	requestDump, err := httputil.DumpRequest(req, false)
	if err != nil {
		return err
	}
	if c.debug {
		slog.Default().Debug("request: \n%s", string(requestDump))
	}

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if c.debug {
		slog.Default().Debug(
			"method=POST url=%s length=%d status=%d duration=%d",
			c.url,
			req.ContentLength,
			resp.StatusCode,
			int(time.Since(start).Milliseconds()),
		)
	}

	responseDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return err
	}
	if c.debug {
		slog.Default().Debug("response: \n%s", string(responseDump))
	}

	if resp.StatusCode != 200 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("pushing timeseries: %d - %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}
