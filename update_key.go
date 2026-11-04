package busbar

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MarshalJSON preserves the absent, null, and string states of the group patch.
func (r UpdateKeyReq) MarshalJSON() ([]byte, error) {
	type plain UpdateKeyReq
	if !r.GroupNull {
		return json.Marshal(plain(r))
	}
	if r.Group != nil {
		return nil, fmt.Errorf("group and GroupNull cannot both be set")
	}
	return json.Marshal(struct {
		plain
		Group *string `json:"group"`
	}{plain: plain(r)})
}

// UnmarshalJSON retains an explicit null instead of treating it as an absent field.
func (r *UpdateKeyReq) UnmarshalJSON(data []byte) error {
	type plain UpdateKeyReq
	parsed := plain(*r)
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	var fields struct {
		Group json.RawMessage `json:"group"`
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if len(fields.Group) != 0 {
		parsed.GroupNull = bytes.Equal(bytes.TrimSpace(fields.Group), []byte("null"))
	}
	*r = UpdateKeyReq(parsed)
	return nil
}
