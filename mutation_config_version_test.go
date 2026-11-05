package busbar

import (
	"encoding/json"
	"testing"
)

func TestFinalSchemaMutationConfigVersionsRetainFullU64(t *testing.T) {
	const token = "18446744073709551615"
	for _, arm := range []string{"legacy", "document"} {
		t.Run(arm, func(t *testing.T) {
			input := []byte(`{"applied":true,"config_version":` + token + `,"settings":{},"configuration":{}}`)
			var union ConfigSettingsMutationView
			if err := json.Unmarshal(input, &union); err != nil {
				t.Fatal(err)
			}
			if arm == "legacy" {
				view, err := union.AsConfigSettingsView()
				if err != nil {
					t.Fatalf("decode legacy config generation: %v", err)
				}
				var version uint64 = view.ConfigVersion
				if version != ^uint64(0) {
					t.Fatalf("legacy config generation rounded: %d", version)
				}
				if err := union.FromConfigSettingsView(view); err != nil {
					t.Fatal(err)
				}
			} else {
				view, err := union.AsConfigDocumentMutationView()
				if err != nil {
					t.Fatalf("decode document config generation: %v", err)
				}
				if _, ok := any(view.ConfigVersion).(uint64); !ok {
					t.Fatalf("document config generation type %T, want uint64", view.ConfigVersion)
				}
				// JSON encoding exercises the generated concrete type while also
				// allowing the previous int mapping to fail causally at decode.
				version, err := json.Marshal(view.ConfigVersion)
				if err != nil || string(version) != token {
					t.Fatalf("document config generation rounded: %s, %v", version, err)
				}
				if err := union.FromConfigDocumentMutationView(view); err != nil {
					t.Fatal(err)
				}
			}
			encoded, err := json.Marshal(union)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields["config_version"]) != token {
				t.Fatalf("mutation union changed config generation: %s", encoded)
			}
		})
	}
}
