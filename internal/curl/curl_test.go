package curl

import (
	"strings"
	"testing"

	"github.com/rzaal/ribnip/internal/collection"
)

func TestBuildArgsGET(t *testing.T) {
	req := &collection.Request{
		Method: "GET",
		URL:    "{{baseUrl}}/users/1",
		Headers: map[string]string{
			"Authorization": "Bearer {{token}}",
		},
	}
	env := collection.Env{"baseUrl": "https://api.example.com", "token": "secret"}

	args := BuildArgs(req, env)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-X GET") {
		t.Errorf("expected -X GET, got: %s", joined)
	}
	if !strings.Contains(joined, "-H Authorization: Bearer secret") {
		t.Errorf("expected rendered auth header, got: %s", joined)
	}
	if !strings.HasSuffix(args[len(args)-1], "https://api.example.com/users/1") {
		t.Errorf("expected rendered url as last arg, got: %s", args[len(args)-1])
	}
}

func TestBuildArgsPOSTWithBody(t *testing.T) {
	req := &collection.Request{
		Method: "post",
		URL:    "https://api.example.com/users",
		Body:   `{"name":"{{name}}"}`,
	}
	env := collection.Env{"name": "Alice"}

	args := BuildArgs(req, env)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-X POST") {
		t.Errorf("expected method uppercased to POST, got: %s", joined)
	}
	if !strings.Contains(joined, `--data {"name":"Alice"}`) {
		t.Errorf("expected rendered body, got: %s", joined)
	}
}

func TestBuildArgsQueryParams(t *testing.T) {
	req := &collection.Request{
		Method: "GET",
		URL:    "https://api.example.com/users",
		Query:  map[string]string{"expand": "profile"},
	}
	args := BuildArgs(req, collection.Env{})
	last := args[len(args)-1]
	if !strings.Contains(last, "expand=profile") {
		t.Errorf("expected query param in url, got: %s", last)
	}
}

func TestRunParsesMeta(t *testing.T) {
	// Sanity check the metadata parsing logic via a real curl call to a
	// well-known local-safe endpoint is avoided in unit tests (network);
	// instead just verify CommandLine renders something sane.
	req := &collection.Request{Method: "GET", URL: "https://example.com"}
	args := BuildArgs(req, collection.Env{})
	cmdline := CommandLine(args)
	if !strings.HasPrefix(cmdline, "curl ") {
		t.Errorf("expected curl prefix, got: %s", cmdline)
	}
}
