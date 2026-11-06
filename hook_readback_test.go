package busbar

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCanonicalHookReadbackPreservesReplacementInputs(t *testing.T) {
	for _, policy := range []string{`null`, `"first"`} {
		body := `{"name":"ranking","kind":"gate","transport":{"kind":"none","target":null},"prompt":"no","user":"no","priority":0,"on_error":"reject","on_empty":` + policy + `,"signals":["requested_model","latency"],"default":true,"timeout_ms":1000,"settings_keys":[],"global":false,"phase":["routing"],"fires_at":["routing"],"groups":[]}`
		var hook HookView
		if err := json.Unmarshal([]byte(body), &hook); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(hook)
		if err != nil {
			t.Fatal(err)
		}
		before, after := additiveJSON(t, []byte(body)), additiveJSON(t, encoded)
		for _, key := range []string{"on_empty", "signals", "default", "phase", "fires_at"} {
			if value, present := after[key]; !present || !reflect.DeepEqual(value, before[key]) {
				t.Fatalf("hook replacement input %s changed: %s", key, encoded)
			}
		}
		if _, present := after["at"]; present {
			t.Fatalf("obsolete at selector added to hook: %s", encoded)
		}
	}
}
