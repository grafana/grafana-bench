package notifier

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

var (
	ErrInvalidMapping        = errors.New("invalid mapping")
	ErrGettingMapping        = errors.New("getting mapping")
	ErrNoMappingForCodeowner = errors.New("no mapping for codeowner")
)

// CodeownersMapping defines the mapping between a code owner and a slack channel
type CodeownersMapping map[string]string

func (d CodeownersMapping) GetChannel(recipient string) (string, error) {
	addr, ok := d[recipient]
	if !ok {
		return "", ErrNoMappingForCodeowner
	}
	return addr, nil
}

// NewCodeownersMapping returns a new CodeownersMapping from the given mapping.
// The mapping's keys should be the names of the teams in the repository and
// the values should be the Slack channels that should receive notifications.
func NewCodeownersMapping(mapping map[string]string) CodeownersMapping {
	return CodeownersMapping(mapping)
}

// MappingEntry defines the mapping between a code owner and a slack channel
type MappingEntry struct {
	GithubTeam   string `yaml:"github_team"`
	SlackChannel string `yaml:"slack_channel"`
}

// YamlMapping defines the structure of a YAML mapping
type YamlMapping struct {
	Mapping []MappingEntry `yaml:"mapping"`
}

// MappingFromFile reads a YAML file from the local filesystem, parses it,
// and returns the corresponding Mapping.
func MappingFromFile(path string) (CodeownersMapping, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w from %q: %w", ErrGettingMapping, path, err)
	}
	defer file.Close()

	mapping, err := MappingFromReader(file)
	if err != nil {
		return nil, fmt.Errorf("%w from %q: %w", ErrGettingMapping, path, err)
	}
	return mapping, nil
}

// MappingFromULR reads a YAML file from a URL, parses it,
// and returns the corresponding Mapping.
func MappingFromULR(url string) (CodeownersMapping, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%w from %q: %w", ErrGettingMapping, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w from %q: %s", ErrGettingMapping, url, resp.Status)
	}

	directory, err := MappingFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w from %q: %w", ErrGettingMapping, url, err)
	}
	return directory, nil
}

// MappingFromReader reads a YAML file from a reader, parses it,
// and returns the corresponding Mapping.
func MappingFromReader(r io.Reader) (CodeownersMapping, error) {
	buffer := new(bytes.Buffer)
	_, err := buffer.ReadFrom(r)
	if err != nil {
		return nil, fmt.Errorf("%w %w", ErrInvalidMapping, err)
	}

	yamlMapping := &YamlMapping{}
	err = yaml.Unmarshal(buffer.Bytes(), yamlMapping)
	if err != nil {
		return nil, fmt.Errorf("%w %w", ErrInvalidMapping, err)
	}

	mapping := map[string]string{}
	for _, entry := range yamlMapping.Mapping {
		if entry.SlackChannel == "" {
			continue
		}
		mapping[entry.GithubTeam] = entry.SlackChannel
	}

	return NewCodeownersMapping(mapping), nil
}
