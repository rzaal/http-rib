// Package capture extracts values from an HTTP response for a request's
// post.captures block, using a small extractor grammar:
//
//	json:<dotted.path[0]>  - walk a dotted path (with optional [n] array
//	                         index) into the JSON response body
//	header:<name>          - a response header (case-insensitive)
//	regex:<expr>           - first capture group of a regex over the body
package capture

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
)

// Response is a parsed HTTP response: headers plus the final body.
type Response struct {
	Headers http.Header
	Body    string
}

// Parse splits combined curl -i output (headers + body, possibly with one
// header block per redirect hop) into a Response. It keeps the last
// headers/body block, since that corresponds to the final response.
func Parse(raw string) Response {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	// curl -i with redirects prints one status-line+headers block per hop,
	// each followed by a blank line. Split on blank lines that immediately
	// follow a header block, keeping the last block as headers and whatever
	// remains as body.
	blocks := strings.Split(raw, "\n\n")

	// Find the last block that looks like a status line (HTTP/...); the
	// header block is that one, everything after it is body.
	headerBlockIdx := -1
	for i, b := range blocks {
		if strings.HasPrefix(b, "HTTP/") {
			headerBlockIdx = i
		}
	}

	if headerBlockIdx == -1 {
		// No recognizable header block; treat everything as body.
		return Response{Headers: http.Header{}, Body: raw}
	}

	headerBlock := blocks[headerBlockIdx]
	body := strings.Join(blocks[headerBlockIdx+1:], "\n\n")

	headers := http.Header{}
	scanner := bufio.NewScanner(strings.NewReader(headerBlock))
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			continue // status line
		}
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		headers.Add(textproto.TrimString(k), textproto.TrimString(v))
	}

	return Response{Headers: headers, Body: body}
}

// Extract applies each capture's extractor against r and returns the
// resolved values. Captures whose extractor can't resolve a value (missing
// header, unmatched regex, absent JSON path) are omitted from the result and
// their names are returned in skipped.
func Extract(captures map[string]string, r Response) (values map[string]string, skipped []string) {
	values = make(map[string]string, len(captures))
	for name, expr := range captures {
		val, ok := extractOne(expr, r)
		if !ok {
			skipped = append(skipped, name)
			continue
		}
		values[name] = val
	}
	return values, skipped
}

func extractOne(expr string, r Response) (string, bool) {
	kind, arg, ok := strings.Cut(expr, ":")
	if !ok {
		return "", false
	}
	switch kind {
	case "json":
		return extractJSON(arg, r.Body)
	case "header":
		v := r.Headers.Get(arg)
		return v, v != ""
	case "regex":
		return extractRegex(arg, r.Body)
	default:
		return "", false
	}
}

func extractRegex(expr, body string) (string, bool) {
	re, err := regexp.Compile(expr)
	if err != nil {
		return "", false
	}
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return "", false
	}
	return m[1], true
}

func extractJSON(path, body string) (string, bool) {
	var data interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return "", false
	}

	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return jsonLeafString(data)
	}

	cur := data
	for _, seg := range splitJSONPath(path) {
		if seg.key != "" {
			m, ok := cur.(map[string]interface{})
			if !ok {
				return "", false
			}
			cur, ok = m[seg.key]
			if !ok {
				return "", false
			}
		}
		if seg.hasIndex {
			arr, ok := cur.([]interface{})
			if !ok || seg.index < 0 || seg.index >= len(arr) {
				return "", false
			}
			cur = arr[seg.index]
		}
	}
	return jsonLeafString(cur)
}

type jsonPathSeg struct {
	key      string
	hasIndex bool
	index    int
}

// splitJSONPath turns "data.items[0].id" into segments: {data} {items,[0]} {id}.
func splitJSONPath(path string) []jsonPathSeg {
	var segs []jsonPathSeg
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			continue
		}
		key := part
		seg := jsonPathSeg{}
		if idx := strings.Index(part, "["); idx != -1 && strings.HasSuffix(part, "]") {
			key = part[:idx]
			n, err := strconv.Atoi(part[idx+1 : len(part)-1])
			if err == nil {
				seg.hasIndex = true
				seg.index = n
			}
		}
		seg.key = key
		segs = append(segs, seg)
	}
	return segs
}

func jsonLeafString(v interface{}) (string, bool) {
	if v == nil {
		return "", false
	}
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(t), true
	case map[string]interface{}, []interface{}:
		return fmt.Sprint(t), true
	default:
		return fmt.Sprint(t), true
	}
}
