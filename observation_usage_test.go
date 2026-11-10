package busbar

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestUsageObservationsPreserveExactStringsAndOmission(t *testing.T) {
	var typed ObservationUsageView
	var _ string = typed.AccountingTime
	var _ string = typed.SpendMicros
	var _ map[string]string = typed.Counts

	observation := map[string]any{
		"key_id": "key-1", "group": "team", "instance": "gateway", "destination": "provider",
		"serving_subject": "requested-model", "rated_subject": "billed-model",
		"accounting_time": "18446744073709551615", "spend_micros": "9223372036854775.807",
		"counts": map[string]any{"duration": "170141183460469231731687303715884.105727", "requests": "0.000001"},
	}
	for _, present := range []bool{true, false} {
		payload := map[string]any{
			"window": map[string]any{"start": 0, "end": 1}, "as_of": 1, "currency": "USD", "rate_card_version": nil,
			"total":    map[string]any{"tokens_input": 0, "tokens_output": 0, "tokens_cache_read": 0, "tokens_cache_creation": 0, "requests": 0, "spend_micros": 0},
			"by_model": []any{}, "by_key": []any{}, "by_key_truncated": false,
		}
		if present {
			payload["observations"] = []any{observation}
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseGetUsageResponse(&http.Response{
			StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(string(body))),
		})
		if err != nil || parsed.JSON200 == nil {
			t.Fatalf("usage response: %v", err)
		}
		encoded, err := json.Marshal(parsed.JSON200)
		if err != nil {
			t.Fatal(err)
		}
		var actual map[string]any
		if err := json.Unmarshal(encoded, &actual); err != nil {
			t.Fatal(err)
		}
		rows, exists := actual["observations"]
		if exists != present {
			t.Fatalf("observations presence = %v, want %v", exists, present)
		}
		if present && !reflect.DeepEqual(rows, []any{observation}) {
			t.Fatalf("observation strings changed: %s", encoded)
		}
	}
}
