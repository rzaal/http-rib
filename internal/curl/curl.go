// Package curl builds curl argv from a request+env and executes the real
// curl binary, capturing its output.
package curl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/rzaal/http-rib/internal/collection"
	"github.com/rzaal/http-rib/internal/render"
)

// delimiter marks the end of the response body / start of curl's -w metadata
// in the combined output stream.
const metaDelim = "\n===HTTP-RIB-META==="

// BuildArgs renders {{vars}} and :params in req against env/params and
// assembles the curl argv (without the leading "curl" itself — that's the
// exec'd binary name).
func BuildArgs(req *collection.Request, env collection.Env, params map[string]string) []string {
	renderedURL := render.ApplyParams(render.Apply(req.URL, env), params)
	renderedHeaders := render.ApplyMap(req.Headers, env)
	renderedQuery := render.ApplyParamsMap(render.ApplyMap(req.Query, env), params)
	renderedBody := render.Apply(req.Body, env)

	full := appendQuery(renderedURL, renderedQuery)

	args := []string{
		"-sS",
		"-i", // include response headers in output
		"-X", strings.ToUpper(req.Method),
	}

	// stable header order for reproducible output/tests
	keys := make([]string, 0, len(renderedHeaders))
	for k := range renderedHeaders {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "-H", fmt.Sprintf("%s: %s", k, renderedHeaders[k]))
	}

	if renderedBody != "" {
		args = append(args, "--data", renderedBody)
	}

	args = append(args, "-w", metaDelim+"\n%{http_code} %{time_total}")
	args = append(args, full)
	return args
}

// appendQuery merges a query param map onto a URL, preserving any query
// string already present in rawURL.
func appendQuery(rawURL string, query map[string]string) string {
	if len(query) == 0 {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // leave malformed URLs untouched; curl will report the error
	}
	q := u.Query()
	// stable order for deterministic output
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		q.Set(k, query[k])
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// CommandLine renders args as a copy-pasteable shell command string
// (quoting each argument), prefixed with "curl".
func CommandLine(args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, "curl")
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	safe := true
	for _, r := range s {
		if !(r == '_' || r == '-' || r == '.' || r == '/' || r == ':' || r == '@' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Result is the outcome of running curl.
type Result struct {
	StatusCode int
	TimeTotal  string
	Body       string // response body (headers + body, per -i) minus our -w meta
	ExitCode   int
	Duration   time.Duration
	Command    string // human-readable curl command that was run
	Err        error  // non-nil on exec failure (curl missing, ctx cancel, etc.)
}

// Run execs the real curl binary with args and parses out the -w metadata
// appended by BuildArgs.
func Run(ctx context.Context, args []string) Result {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "curl", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	dur := time.Since(start)

	res := Result{
		Duration: dur,
		Command:  CommandLine(args),
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.Err = fmt.Errorf("run curl: %w", err)
			return res
		}
	}

	raw := out.String()
	body := raw
	if idx := strings.LastIndex(raw, metaDelim); idx != -1 {
		body = raw[:idx]
		meta := strings.TrimSpace(raw[idx+len(metaDelim):])
		var code int
		var timeTotal string
		if _, scanErr := fmt.Sscanf(meta, "%d %s", &code, &timeTotal); scanErr == nil {
			res.StatusCode = code
			res.TimeTotal = timeTotal
		}
	}
	res.Body = body
	return res
}

// CheckAvailable errors with a clear message if curl isn't on PATH.
func CheckAvailable() error {
	if _, err := exec.LookPath("curl"); err != nil {
		return fmt.Errorf("curl not found on PATH: install curl to use http-rib")
	}
	return nil
}
