package render

import (
	"reflect"
	"testing"
)

func TestExtractParamNames(t *testing.T) {
	got := ExtractParamNames("https://api.example.com/users/:id/orders/:orderId", map[string]string{
		"filter": ":status",
		"sort":   "name",
	})
	want := []string{"id", "orderId", "status"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExtractParamNames() = %v, want %v", got, want)
	}
}

func TestExtractParamNamesDedup(t *testing.T) {
	got := ExtractParamNames("/users/:id/friends/:id", nil)
	want := []string{"id"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExtractParamNames() = %v, want %v", got, want)
	}
}

func TestApplyParams(t *testing.T) {
	params := map[string]string{"id": "42"}

	got := ApplyParams("/users/:id", params)
	if got != "/users/42" {
		t.Errorf("ApplyParams() = %q", got)
	}

	// unset param left as-is
	got = ApplyParams("/users/:missing", params)
	if got != "/users/:missing" {
		t.Errorf("ApplyParams() = %q", got)
	}
}

func TestApplyParamsMap(t *testing.T) {
	params := map[string]string{"status": "active"}
	m := map[string]string{"filter": ":status"}
	got := ApplyParamsMap(m, params)
	if got["filter"] != "active" {
		t.Errorf("ApplyParamsMap() = %v", got)
	}
}
