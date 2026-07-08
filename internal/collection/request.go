package collection

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Request is one HTTP request definition loaded from a YAML file.
type Request struct {
	Name    string            `yaml:"name"`
	Method  string            `yaml:"method"`
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
	Query   map[string]string `yaml:"query"`
	Body    string            `yaml:"body"`

	// Path is the absolute path to the source file (not serialized).
	Path string `yaml:"-"`
}

func LoadRequest(path string) (*Request, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read request %s: %w", path, err)
	}
	var r Request
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse request %s: %w", path, err)
	}
	if r.Method == "" {
		r.Method = "GET"
	}
	r.Path = path
	return &r, nil
}
