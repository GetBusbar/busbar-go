package busbar

import (
	"encoding/json"
	"io"
	"testing"
)

// This checks the SDK request codec, not whether these probe settings are valid
// configuration. Backend integer-token rules must receive the caller's tokens.
func TestFinalSchemaSettingsNumericTokensRemainUnchanged(t *testing.T) {
	const input = `{"document_patch":{"set":{"/probe/unsigned":18446744073709551615,"/probe/signed":-9223372036854775808,"/probe/count":9007199254.740993,"/probe/fractional":1.0,"/probe/exponent":1e3}}}`
	var body PutConfigSettingsJSONRequestBody
	if err := json.Unmarshal([]byte(input), &body); err != nil {
		t.Fatal(err)
	}
	request, err := NewPutConfigSettingsRequest("https://example.test/", nil, body)
	if err != nil {
		t.Fatal(err)
	}
	defer request.Body.Close()
	raw, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		DocumentPatch struct {
			Set map[string]json.RawMessage `json:"set"`
		} `json:"document_patch"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for path, token := range map[string]string{
		"/probe/unsigned":   "18446744073709551615",
		"/probe/signed":     "-9223372036854775808",
		"/probe/count":      "9007199254.740993",
		"/probe/fractional": "1.0",
		"/probe/exponent":   "1e3",
	} {
		if string(decoded.DocumentPatch.Set[path]) != token {
			t.Errorf("%s token changed: got %s, want %s", path, decoded.DocumentPatch.Set[path], token)
		}
	}
}
