package capture

import "testing"

func TestParseSimple(t *testing.T) {
	raw := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nX-Csrf-Token: abc123\r\n\r\n{\"data\":{\"token\":\"sess-1\"}}"
	r := Parse(raw)
	if got := r.Headers.Get("X-Csrf-Token"); got != "abc123" {
		t.Fatalf("header X-Csrf-Token = %q, want abc123", got)
	}
	if r.Body != `{"data":{"token":"sess-1"}}` {
		t.Fatalf("body = %q", r.Body)
	}
}

func TestParseRedirectKeepsLastBlock(t *testing.T) {
	raw := "HTTP/1.1 302 Found\r\nLocation: /next\r\n\r\nHTTP/1.1 200 OK\r\nX-Final: yes\r\n\r\n{\"ok\":true}"
	r := Parse(raw)
	if got := r.Headers.Get("X-Final"); got != "yes" {
		t.Fatalf("header X-Final = %q, want yes (expected last hop's headers)", got)
	}
	if got := r.Headers.Get("Location"); got != "" {
		t.Fatalf("header Location = %q, want empty (should not leak first hop's headers)", got)
	}
	if r.Body != `{"ok":true}` {
		t.Fatalf("body = %q", r.Body)
	}
}

func TestExtractJSON(t *testing.T) {
	r := Response{Body: `{"data":{"token":"sess-1","items":[{"id":7},{"id":8}]}}`}
	values, skipped := Extract(map[string]string{
		"sessionId": "json:.data.token",
		"secondID":  "json:.data.items[1].id",
	}, r)
	if len(skipped) != 0 {
		t.Fatalf("unexpected skipped: %v", skipped)
	}
	if values["sessionId"] != "sess-1" {
		t.Fatalf("sessionId = %q", values["sessionId"])
	}
	if values["secondID"] != "8" {
		t.Fatalf("secondID = %q", values["secondID"])
	}
}

func TestExtractHeader(t *testing.T) {
	r := Response{Headers: map[string][]string{"X-Csrf-Token": {"abc123"}}}
	values, skipped := Extract(map[string]string{"csrf": "header:X-Csrf-Token"}, r)
	if len(skipped) != 0 {
		t.Fatalf("unexpected skipped: %v", skipped)
	}
	if values["csrf"] != "abc123" {
		t.Fatalf("csrf = %q", values["csrf"])
	}
}

func TestExtractRegex(t *testing.T) {
	r := Response{Body: "order=abc123&status=ok"}
	values, skipped := Extract(map[string]string{"orderId": `regex:order=(\w+)`}, r)
	if len(skipped) != 0 {
		t.Fatalf("unexpected skipped: %v", skipped)
	}
	if values["orderId"] != "abc123" {
		t.Fatalf("orderId = %q", values["orderId"])
	}
}

func TestExtractSkipsMissing(t *testing.T) {
	r := Response{Body: `{"data":{}}`}
	values, skipped := Extract(map[string]string{
		"missing": "json:.data.token",
		"noHdr":   "header:X-Nope",
		"noMatch": "regex:zzz(\\w+)",
	}, r)
	if len(values) != 0 {
		t.Fatalf("expected no values, got %v", values)
	}
	if len(skipped) != 3 {
		t.Fatalf("expected 3 skipped, got %v", skipped)
	}
}
