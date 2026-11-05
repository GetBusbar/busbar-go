package busbar

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefaultUsageResponsePreservesExactCounts(t *testing.T) {
	for _, count := range []string{
		"0.1", "27.5", "9007199254740993", "9223372036854.775807",
		"170141183460469231731687303715884.105727",
	} {
		t.Run(count, func(t *testing.T) {
			body := strings.ReplaceAll(`{"as_of":1700000000,"currency":"USD","window":{"start":1699920000,"end":1700006400},"rate_card_version":"0123456789abcdef0123456789abcdef01234567","by_key":[],"by_key_truncated":false,"by_model":[{"model":"model","provider":"provider","requests":3,"spend_micros":4200,"tokens_input":COUNT,"tokens_output":COUNT,"tokens_cache_creation":COUNT,"tokens_cache_read":COUNT}],"total":{"requests":3,"spend_micros":4200,"tokens_input":COUNT,"tokens_output":COUNT,"tokens_cache_creation":COUNT,"tokens_cache_read":COUNT}}`, "COUNT", count)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/admin/usage" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, body)
			}))
			defer server.Close()
			client, err := NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.GetUsageWithResponse(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if response.JSON200 == nil {
				t.Fatalf("missing typed usage response: %s", response.Body)
			}
			usage := response.JSON200
			for _, value := range []any{
				usage.Total.TokensInput, usage.Total.TokensOutput,
				usage.Total.TokensCacheCreation, usage.Total.TokensCacheRead,
				usage.ByModel[0].TokensInput, usage.ByModel[0].TokensOutput,
				usage.ByModel[0].TokensCacheCreation, usage.ByModel[0].TokensCacheRead,
			} {
				number, ok := value.(json.Number)
				if !ok || number.String() != count {
					t.Fatalf("Count decoded as %T(%v), want json.Number(%s)", value, value, count)
				}
				encoded, err := json.Marshal(number)
				if err != nil || string(encoded) != count {
					t.Fatalf("Count wire roundtrip = %s, %v; want numeric %s", encoded, err, count)
				}
			}
			if usage.Total.SpendMicros != 4200 || usage.Total.Requests != 3 {
				t.Fatalf("unrelated money/counter changed: %+v", usage.Total)
			}
			encoded, err := json.Marshal(usage)
			if err != nil {
				t.Fatal(err)
			}
			var roundtrip struct {
				Total map[string]json.RawMessage `json:"total"`
			}
			if err := json.Unmarshal(encoded, &roundtrip); err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"tokens_input", "tokens_output", "tokens_cache_creation", "tokens_cache_read"} {
				if string(roundtrip.Total[field]) != count {
					t.Fatalf("%s encoded as %s, want numeric %s", field, roundtrip.Total[field], count)
				}
			}
		})
	}
}

func TestUsageErrorEnvelopesRemainTyped(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusServiceUnavailable} {
		body := `{"error":{"code":"invalid_request","message":"invalid history entry key"}}`
		response, err := ParseGetUsageResponse(&http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		})
		if err != nil {
			t.Fatal(err)
		}
		var envelope *Error
		switch status {
		case http.StatusBadRequest:
			envelope = response.JSON400
		case http.StatusUnauthorized:
			envelope = response.JSON401
		case http.StatusServiceUnavailable:
			envelope = response.JSON503
		}
		if response.JSON200 != nil || envelope == nil || envelope.Error.Code != "invalid_request" || envelope.Error.Message != "invalid history entry key" {
			t.Fatalf("status %d lost typed error: %+v", status, response)
		}
		if response.StatusCode() != status || string(response.Body) != body {
			t.Fatalf("status %d lost original response", status)
		}
	}
}

func TestUsageMalformedCountRetainsHTTPContext(t *testing.T) {
	original := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"total":{"tokens_input":true}}`)),
	}
	_, err := ParseGetUsageResponse(original)
	var parse *ResponseParseError
	if !errors.As(err, &parse) || parse.HTTPResponse != original {
		t.Fatalf("malformed Count lost response context: %v", err)
	}
}

func TestCountKeyAndGroupUsageResponses(t *testing.T) {
	response := func(body string) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}
	}
	key, err := ParseGetKeysIdUsageResponse(response(`{"id":"key","tokens":27.5,"rate_headroom":0.25,"requests":1,"spend_cents":2}`))
	if err != nil || key.JSON200 == nil {
		t.Fatalf("key usage failed: %v", err)
	}
	count, ok := any(key.JSON200.Tokens).(json.Number)
	if !ok || count.String() != "27.5" || key.JSON200.RateHeadroom == nil || *key.JSON200.RateHeadroom != 0.25 {
		t.Fatalf("key Count or unrelated ratio changed: %+v", key.JSON200)
	}
	group, err := ParseGetGroupsNameUsageResponse(response(`{"name":"group","as_of":1,"buckets":[{"window":"total","tokens":9007199254740993.125,"requests":1,"spend_cents":2,"tokens_cap":42}]}`))
	if err != nil || group.JSON200 == nil || len(group.JSON200.Buckets) != 1 {
		t.Fatalf("group usage failed: %v", err)
	}
	bucket := group.JSON200.Buckets[0]
	count, ok = any(bucket.Tokens).(json.Number)
	if !ok || count.String() != "9007199254740993.125" || bucket.TokensCap == nil || *bucket.TokensCap != 42 {
		t.Fatalf("group Count or unrelated integer cap changed: %+v", bucket)
	}
}
