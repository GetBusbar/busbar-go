package busbar

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDefaultClientDoesNotForwardAdminTokenAcrossOrigins(t *testing.T) {
	received := make(chan string, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Get("x-admin-token")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer source.Close()
	client, err := NewClient(source.URL, WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		req.Header.Set("x-admin-token", "secret")
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.GetInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	select {
	case credential := <-received:
		t.Fatalf("cross-origin redirect followed with credential %q", credential)
	default:
	}
	if response.StatusCode != http.StatusFound {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func TestParseResponseRejectsOversizedBody(t *testing.T) {
	body := &countingBody{remaining: (64 << 20) + 1}
	response := &http.Response{StatusCode: 502, Header: make(http.Header), Body: body}
	_, err := ParseGetInfoResponse(response)
	if err == nil {
		t.Fatal("oversized response accepted")
	}
	if !body.closed {
		t.Fatal("response body not closed")
	}
}

type countingBody struct {
	remaining int64
	closed    bool
}

func (b *countingBody) Read(p []byte) (int, error) {
	if b.remaining == 0 {
		return 0, io.EOF
	}
	n := len(p)
	if int64(n) > b.remaining {
		n = int(b.remaining)
	}
	for i := 0; i < n; i++ {
		p[i] = 'x'
	}
	b.remaining -= int64(n)
	return n, nil
}
func (b *countingBody) Close() error { b.closed = true; return nil }

func TestDefaultClientAllowsSameOriginRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/info") {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		if r.Header.Get("x-admin-token") != "secret" {
			t.Error("credential missing on same-origin redirect")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, WithRequestEditorFn(func(_ context.Context, r *http.Request) error { r.Header.Set("x-admin-token", "secret"); return nil }))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.GetInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func TestParseResponseRecognizesJSONMediaType(t *testing.T) {
	for _, contentType := range []string{"application/JSON", "APPLICATION/JSON; charset=utf-8", "application/problem+json"} {
		original := &http.Response{StatusCode: 401, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(`{"error":{"code":"unauthorized","message":"denied"}}`))}
		response, err := ParseGetInfoResponse(original)
		if err != nil {
			t.Fatal(err)
		}
		if response.JSON401 == nil {
			t.Errorf("did not parse %q", contentType)
		}
		if response.HTTPResponse != original || original.Header.Get("Content-Type") != contentType {
			t.Fatal("original response changed")
		}
	}
}

func TestRedirectDefaultPortIsSameOrigin(t *testing.T) {
	for _, pair := range [][2]string{{"https://example.test", "https://EXAMPLE.test:443/final"}, {"http://example.test:80", "http://example.test/final"}} {
		from, _ := url.Parse(pair[0])
		to, _ := url.Parse(pair[1])
		err := newDefaultHTTPClient().CheckRedirect(&http.Request{URL: to}, []*http.Request{{URL: from}})
		if err != nil {
			t.Errorf("same origin %v rejected: %v", pair, err)
		}
	}
}

func TestParseResponseIgnoresJSONInNonJSONMediaType(t *testing.T) {
	for _, contentType := range []string{"text/plain; note=json", "application/notjson"} {
		original := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader("not JSON"))}
		response, err := ParseGetInfoResponse(original)
		if err != nil {
			t.Fatalf("%q: %v", contentType, err)
		}
		if response.JSON200 != nil || string(response.Body) != "not JSON" || response.HTTPResponse != original {
			t.Fatalf("unexpected response for %q", contentType)
		}
	}
}

func TestMalformedJSONRetainsHTTPContext(t *testing.T) {
	original := &http.Response{StatusCode: 401, Header: http.Header{"Content-Type": []string{"application/json"}, "Retry-After": []string{"10"}}, Body: io.NopCloser(strings.NewReader("{"))}
	_, err := ParseGetInfoResponse(original)
	if err == nil {
		t.Fatal("malformed JSON accepted")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("HTTP status missing: %v", err)
	}
	var parse *ResponseParseError
	if !errors.As(err, &parse) || parse.HTTPResponse != original || parse.HTTPResponse.Header.Get("Retry-After") != "10" {
		t.Fatalf("HTTP metadata lost: %v", err)
	}
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatalf("JSON cause lost: %v", err)
	}
}
