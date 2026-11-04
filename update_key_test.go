package busbar

import (
	"encoding/json"
	"io"
	"testing"
)

func TestUpdateKeyGroupThreeStates(t *testing.T) {
	for _, input := range []string{`{}`, `{"group":null}`, `{"group":"team"}`} {
		var request UpdateKeyReq
		if err := json.Unmarshal([]byte(input), &request); err != nil {
			t.Fatal(err)
		}
		output, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		if string(output) != input {
			t.Errorf("round trip %s became %s", input, output)
		}
	}
}

func TestUpdateKeyUnmarshalPreservesAbsentFields(t *testing.T) {
	enabled := true
	request := UpdateKeyReq{Enabled: &enabled, GroupNull: true}
	if err := json.Unmarshal([]byte(`{}`), &request); err != nil {
		t.Fatal(err)
	}
	if request.Enabled != &enabled || !request.GroupNull {
		t.Fatal("absent fields changed existing values")
	}
	if err := json.Unmarshal([]byte(`{"group":"new"}`), &request); err != nil {
		t.Fatal(err)
	}
	if request.GroupNull || request.Group == nil || *request.Group != "new" {
		t.Fatal("string did not replace explicit null")
	}
}

func TestUpdateKeyUnbindRequest(t *testing.T) {
	enabled := false
	request, err := NewPatchKeysIdRequest("https://example.test/", "key", nil, UpdateKeyReq{Enabled: &enabled, GroupNull: true})
	if err != nil {
		t.Fatal(err)
	}
	defer request.Body.Close()
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]interface{}
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	group, present := fields["group"]
	if !present || group != nil || fields["enabled"] != false {
		t.Fatalf("unexpected patch: %s", body)
	}
	name := "team"
	if _, err := json.Marshal(UpdateKeyReq{Group: &name, GroupNull: true}); err == nil {
		t.Fatal("contradictory group settings accepted")
	}
}
