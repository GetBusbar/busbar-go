package busbar

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func requireExactGenericNumber(t *testing.T, value any, token string) {
	t.Helper()
	number, ok := value.(json.Number)
	if !ok || number.String() != token {
		t.Fatalf("generic numeric value is %T(%v), want exact json.Number(%s)", value, value, token)
	}
}

func TestFinalSchemaConfigAccessorsDecodeEditReencode(t *testing.T) {
	for _, mode := range []string{"legacy", "document"} {
		t.Run(mode, func(t *testing.T) {
			// RootSettings is a closed object. Arbitrary module-owned keys belong
			// inside store.settings; document mode addresses them by JSON Pointer.
			input := `{"store":{"module":"test-store","settings":{"u64":18446744073709551615,"decimal":9007199254.740993,"signed":-9223372036854775808}}}`
			prefix := ""
			if mode == "document" {
				prefix = "/store/settings/"
				input = `{"document_patch":{"set":{"/store/settings/u64":18446744073709551615,"/store/settings/decimal":9007199254.740993,"/store/settings/signed":-9223372036854775808}}}`
			}
			var body PutConfigSettingsJSONRequestBody
			if err := json.Unmarshal([]byte(input), &body); err != nil {
				t.Fatal(err)
			}
			var edited map[string]interface{}
			if mode == "legacy" {
				view, err := body.AsPutConfigSettingsJSONBody0()
				if err != nil || view.Store == nil {
					t.Fatalf("store accessor: %v", err)
				}
				store, err := view.Store.AsPutConfigSettingsJSONBody0Store0()
				if err != nil {
					t.Fatal(err)
				}
				edited = store["settings"].(map[string]interface{})
				edited["edited"] = true
				if err := view.Store.FromPutConfigSettingsJSONBody0Store0(store); err != nil {
					t.Fatal(err)
				}
				if err := body.FromPutConfigSettingsJSONBody0(view); err != nil {
					t.Fatal(err)
				}
			} else {
				view, err := body.AsPutConfigSettingsJSONBody1()
				if err != nil || view.DocumentPatch.Set == nil {
					t.Fatalf("document accessor: %v", err)
				}
				edited = *view.DocumentPatch.Set
				edited[prefix+"edited"] = true
				if err := body.FromPutConfigSettingsJSONBody1(view); err != nil {
					t.Fatal(err)
				}
			}
			requireExactGenericNumber(t, edited[prefix+"u64"], "18446744073709551615")
			requireExactGenericNumber(t, edited[prefix+"decimal"], "9007199254.740993")
			requireExactGenericNumber(t, edited[prefix+"signed"], "-9223372036854775808")
			request, err := NewPutConfigSettingsRequest("https://example.test/", nil, body)
			if err != nil {
				t.Fatal(err)
			}
			defer request.Body.Close()
			raw, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}
			var object map[string]json.RawMessage
			if mode == "document" {
				var document struct {
					DocumentPatch struct {
						Set map[string]json.RawMessage `json:"set"`
					} `json:"document_patch"`
				}
				if err := json.Unmarshal(raw, &document); err != nil {
					t.Fatal(err)
				}
				object = document.DocumentPatch.Set
			} else {
				var legacy struct {
					Store struct {
						Settings map[string]json.RawMessage `json:"settings"`
					} `json:"store"`
				}
				if err := json.Unmarshal(raw, &legacy); err != nil {
					t.Fatal(err)
				}
				object = legacy.Store.Settings
			}
			for key, token := range map[string]string{"u64": "18446744073709551615", "decimal": "9007199254.740993", "signed": "-9223372036854775808", "edited": "true"} {
				if string(object[prefix+key]) != token {
					t.Errorf("%s changed on actual request wire: %s, want %s", key, object[prefix+key], token)
				}
			}
		})
	}
}

func TestFinalSchemaConfigResponseGenericNumbersRemainExact(t *testing.T) {
	const body = `{"applied":false,"config_version":7,"settings":{"u64":18446744073709551615,"decimal":9007199254.740993},"configuration":{"counter":18446744073709551615}}`
	parsed, err := ParseGetConfigSettingsResponse(&http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))})
	if err != nil || parsed.JSON200 == nil {
		t.Fatalf("configuration response: %v", err)
	}
	settings, ok := parsed.JSON200.Settings.(map[string]interface{})
	if !ok {
		t.Fatalf("settings type %T", parsed.JSON200.Settings)
	}
	requireExactGenericNumber(t, settings["u64"], "18446744073709551615")
	requireExactGenericNumber(t, settings["decimal"], "9007199254.740993")
	if parsed.JSON200.Configuration == nil {
		t.Fatal("configuration disappeared")
	}
	requireExactGenericNumber(t, (*parsed.JSON200.Configuration)["counter"], "18446744073709551615")
	settings["edited"] = true
	encoded, err := json.Marshal(parsed.JSON200)
	if err != nil || !strings.Contains(string(encoded), `"u64":18446744073709551615`) || !strings.Contains(string(encoded), `"decimal":9007199254.740993`) {
		t.Fatalf("response edit/re-encode rounded generic values: %s, %v", encoded, err)
	}
}

func TestFinalSchemaLosslessDecoderKeepsInvalidJSONErrors(t *testing.T) {
	for _, input := range []string{"", "{} {}", "{"} {
		var value any
		err := unmarshalJSONExact([]byte(input), &value)
		var syntax *json.SyntaxError
		if !errors.As(err, &syntax) {
			t.Errorf("invalid JSON %q lost json.Unmarshal syntax error: %v", input, err)
		}
	}
}
