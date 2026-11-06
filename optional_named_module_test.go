package busbar

import (
	"encoding/json"
	"testing"
)

func TestNamedDefinitionOptionalModuleRoundTrip(t *testing.T) {
	for _, body := range []string{
		`{"items":[{"name":"module-free-plane","settings_keys":[]}],"next_cursor":null}`,
		`{"items":[{"name":"plugin-plane","module":"llm","settings_keys":[]}],"next_cursor":null}`,
	} {
		parsed, err := ParseGetAgentsResponse(finalSchemaResponse(body))
		if err != nil || parsed.JSON200 == nil {
			t.Fatalf("named definition response: %v", err)
		}
		encoded, err := json.Marshal(parsed.JSON200)
		if err != nil {
			t.Fatal(err)
		}
		var original, roundTrip map[string]any
		if err := json.Unmarshal([]byte(body), &original); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(encoded, &roundTrip); err != nil {
			t.Fatal(err)
		}
		before := original["items"].([]any)[0].(map[string]any)
		after := roundTrip["items"].([]any)[0].(map[string]any)
		want, wantPresent := before["module"]
		got, gotPresent := after["module"]
		if gotPresent != wantPresent || got != want {
			t.Fatalf("module omission or value changed: input %s, round trip %s", body, encoded)
		}
	}
}
