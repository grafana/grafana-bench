// Package recorder implements API call recording
package recorder

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	DefaultProxyPort = 8080
	Binary           = "/usr/bin/mitmdump"
)

var (
	ErrInvalidOptions   = errors.New("invalid options")
	ErrStartProxy       = errors.New("error starting proxy")
	ErrStoppingProxy    = errors.New("error stopping proxy")
	ErrParsingRecording = errors.New("error parsing recording")
)

type Request struct {
	Host   string
	Method string
	Path   string
	Status string
}
type Recording struct {
	Requests []Request
}

// Recorder is an API call recorder
type Recorder interface {
	GetRecording() (Recording, error)
}

type ProxyRecorder struct {
	url      string
	host     string
	cmd     *exec.Cmd
	capture *bytes.Buffer
}

type ProxyOptions struct {
	GracePeriod time.Duration
	Scheme      string
	Address     string
	Port        int
	Target      string
	Verbose     bool
}

func NewProxyRecorder(opts ProxyOptions) (*ProxyRecorder, error) {
	if opts.Target == "" {
		return nil, fmt.Errorf("%w: target cannot be empty", ErrInvalidOptions)
	}

	targetUrl, err := url.Parse(opts.Target)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOptions, err)
	}

	if opts.Port == 0 {
		opts.Port = DefaultProxyPort
	}

	if opts.Scheme == "" {
		opts.Scheme = "http"
	}
	if opts.Address == "" {
		opts.Address = "127.0.0.1"
	}

	if opts.GracePeriod == 0 {
		opts.GracePeriod = time.Second
	}

	args := []string{
		"--listen-port",
		fmt.Sprintf("%d", opts.Port),
		"--ssl-insecure",
		"--set", "flow_detail=1",
		"--set", "termlog_verbosity=ERROR",
		"--set", fmt.Sprintf("dumper-filter='~d %s'", targetUrl.Hostname()),
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command(Binary, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if opts.Verbose {
		cmd.Stdout = io.MultiWriter(stdout, os.Stdout)
		cmd.Stderr = io.MultiWriter(stderr, os.Stderr)
	}

	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrStartProxy, err)
	}

	time.Sleep(opts.GracePeriod)
	err = cmd.Process.Signal(syscall.Signal(0))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrStartProxy, stderr.String())
	}

	return &ProxyRecorder{
		url:     fmt.Sprintf("%s://%s:%d", opts.Scheme, opts.Address, opts.Port),
		host:    fmt.Sprintf("%s:%d", opts.Address, opts.Port),
		cmd:     cmd,
		capture: stdout,
	}, nil
}

func (p *ProxyRecorder) GetRecording() (Recording, error) {
	//stop the proxy
	err := p.cmd.Process.Kill()
	if err != nil {
		return Recording{}, fmt.Errorf("%w: %s", ErrStoppingProxy, err)
	}

	return ParseRecording(p.capture.Bytes())
}

// ProxyHost returns the host and port of the proxy used to capture the requests
func (p *ProxyRecorder) ProxyHost() string {
	return p.host
}

// ProxyURL returns the irl of the proxy used to capture the requests
func (p *ProxyRecorder) ProxyURL() string {
	return p.url
}

// ParseRecording parses the recording output and returns a Recording
func ParseRecording(buffer []byte) (Recording, error) {
	recording := Recording{}

	scanner := bufio.NewScanner(bytes.NewReader(buffer))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// assume the recording always starts with request line
		// Example: 127.0.0.1:40288: POST https://instance.grafana.net/login HTTP/2.0
		// the HTTP/<version> is optional
		tokens := strings.Split(line, " ")
		if len(tokens) < 3 {
			return Recording{}, fmt.Errorf("%w: %s", ErrParsingRecording, line)
		}

		requestURL, err := url.Parse(tokens[2])
		if err != nil {
			return Recording{}, fmt.Errorf("%w: %s", ErrParsingRecording, line)
		}
		request := Request{
			Host:   requestURL.Host,
			Path:   requestURL.Path,
			Method: tokens[1],
		}

		if scanner.Scan() {
			// parse response (has leading spaces)
			// Example:	<< HTTP/2.0 200 OK 65b
			// the HTTP/<version> is optional

			line = strings.TrimLeft(scanner.Text(), " ")
			tokens = strings.Split(line, " ")
			if len(tokens) < 3 {
				return Recording{}, fmt.Errorf("%w: %s", ErrParsingRecording, line)
			}
			if tokens[0] != "<<" {
				return Recording{}, fmt.Errorf("%w: %s", ErrParsingRecording, line)
			}

			if !strings.HasPrefix(tokens[1], "HTTP") {
				request.Status = tokens[1]
			} else {
				request.Status = tokens[2]
			}
		}
		recording.Requests = append(recording.Requests, request)
	}
	return recording, nil
}
