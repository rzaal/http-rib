package render

import (
	"testing"

	"github.com/rzaal/ribnip/internal/collection"
)

func TestApply(t *testing.T) {
	env := collection.Env{"baseUrl": "https://api.example.com", "token": "secret123"}

	got := Apply("{{baseUrl}}/users/1", env)
	want := "https://api.example.com/users/1"
	if got != want {
		t.Errorf("Apply() = %q, want %q", got, want)
	}

	got = Apply("Bearer {{token}}", env)
	want = "Bearer secret123"
	if got != want {
		t.Errorf("Apply() = %q, want %q", got, want)
	}

	// unknown var left as-is
	got = Apply("{{missing}}", env)
	want = "{{missing}}"
	if got != want {
		t.Errorf("Apply() = %q, want %q", got, want)
	}
}

func TestApplyMap(t *testing.T) {
	env := collection.Env{"token": "abc"}
	m := map[string]string{"Authorization": "Bearer {{token}}"}
	got := ApplyMap(m, env)
	if got["Authorization"] != "Bearer abc" {
		t.Errorf("ApplyMap() = %v", got)
	}
}
