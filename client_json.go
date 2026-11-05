package busbar

import (
	"bytes"
	"encoding/json"
)

// unmarshalJSONExact preserves number tokens in generic JSON values. Check
// syntax first to retain json.Unmarshal's single-document and invalid-input
// errors, including for union types with their own UnmarshalJSON methods.
func unmarshalJSONExact(input []byte, output any) error {
	if !json.Valid(input) {
		return json.Unmarshal(input, output)
	}
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	return decoder.Decode(output)
}
