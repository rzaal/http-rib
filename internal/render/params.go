// Package render: :param placeholder support for URL paths and query values.
package render

import (
	"regexp"
	"sort"
)

var paramPattern = regexp.MustCompile(`:([A-Za-z_]\w*)`)

// ExtractParamNames returns the ordered, deduplicated list of :name params
// referenced in a request's URL and query map. URL params come first (in
// occurrence order), followed by query params (in key-sorted order), so the
// tab order is stable across renders.
func ExtractParamNames(url string, query map[string]string) []string {
	seen := make(map[string]bool)
	var out []string

	add := func(s string) {
		for _, m := range paramPattern.FindAllStringSubmatch(s, -1) {
			name := m[1]
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}

	add(url)

	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		add(query[k])
	}

	return out
}

// ApplyParams replaces every :name occurrence in s with params[name]. Params
// with no stored value (or unknown names) are left as-is.
func ApplyParams(s string, params map[string]string) string {
	return paramPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := paramPattern.FindStringSubmatch(match)
		name := sub[1]
		if val, ok := params[name]; ok && val != "" {
			return val
		}
		return match
	})
}

// ApplyParamsMap runs ApplyParams over every value in a map.
func ApplyParamsMap(m map[string]string, params map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = ApplyParams(v, params)
	}
	return out
}
