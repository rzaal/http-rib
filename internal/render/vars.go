// Package render substitutes {{var}} placeholders using environment values.
package render

import (
	"regexp"

	"github.com/rzaal/ribnip/internal/collection"
)

var varPattern = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

// Apply replaces every {{name}} occurrence in s with env[name]. Unknown
// variables are left as-is (visibly unresolved) rather than silently dropped.
func Apply(s string, env collection.Env) string {
	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := varPattern.FindStringSubmatch(match)
		name := sub[1]
		if val, ok := env[name]; ok {
			return val
		}
		return match
	})
}

// ApplyMap runs Apply over every value in a map, returning a new map.
func ApplyMap(m map[string]string, env collection.Env) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = Apply(v, env)
	}
	return out
}
