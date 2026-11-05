package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDeriveCountSchema(t *testing.T) {
	input := []byte(`{"components":{"schemas":{"Usage":{"properties":{"count":{"type":"number","x-busbar-count-scale":6,"maximum":170141183460469231731687303715884.105727},"ratio":{"type":"number"},"requests":{"type":"integer"},"nested":{"items":{"type":"number","x-busbar-count-scale":6}}}}}}}`)
	original := bytes.Clone(input)
	derived, err := deriveSchema(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(input, original) {
		t.Fatal("canonical schema input was modified")
	}
	if !bytes.Contains(derived, []byte(`170141183460469231731687303715884.105727`)) {
		t.Fatalf("schema bound lost precision: %s", derived)
	}
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					GoType   string            `json:"x-go-type"`
					GoImport map[string]string `json:"x-go-type-import"`
					Items    struct {
						GoType string `json:"x-go-type"`
					} `json:"items"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(derived, &doc); err != nil {
		t.Fatal(err)
	}
	properties := doc.Components.Schemas["Usage"].Properties
	if properties["count"].GoType != "json.Number" || properties["count"].GoImport != nil {
		t.Fatalf("Count mapping missing: %+v", properties["count"])
	}
	for _, name := range []string{"ratio", "requests"} {
		if properties[name].GoType != "" || properties[name].GoImport != nil {
			t.Fatalf("unannotated property %s was changed", name)
		}
	}
	if properties["nested"].Items.GoType != "json.Number" {
		t.Fatal("nested Count mapping missing")
	}
	repeated, err := deriveSchema(derived)
	if err != nil || !bytes.Equal(derived, repeated) {
		t.Fatalf("adapter is not deterministic/idempotent: %v", err)
	}
}

func TestDeriveCountSchemaRejectsUnknownScale(t *testing.T) {
	for _, scale := range []string{`7`, `"6"`, `null`} {
		_, err := deriveSchema([]byte(`{"type":"number","x-busbar-count-scale":` + scale + `}`))
		if err == nil || !strings.Contains(err.Error(), "unsupported x-busbar-count-scale") {
			t.Fatalf("scale %s: %v", scale, err)
		}
	}
}

func TestDeriveCountSchemaRejectsNonnumericWireType(t *testing.T) {
	if _, err := deriveSchema([]byte(`{"type":"string","x-busbar-count-scale":6}`)); err == nil {
		t.Fatal("Count annotation must not change the numeric wire contract")
	}
}

func TestDeriveCountSchemaRejectsInvalidJSON(t *testing.T) {
	for _, input := range []string{`{`, `{} {}`, `{} garbage`} {
		if _, err := deriveSchema([]byte(input)); err == nil {
			t.Fatalf("accepted invalid document %q", input)
		}
	}
}

func TestDeriveCountSchemaExactNumberMetadata(t *testing.T) {
	input := []byte(`{"properties":{"unsigned":{"type":"integer","x-busbar-exact-number":{"bits":64,"signed":false}},"nullable":{"type":["integer","null"],"format":"uint","x-busbar-exact-number":{"integer_format":"uint"}},"signed":{"type":"integer","x-busbar-exact-number":{"integer_format":"int64"}},"count":{"anyOf":[{"type":"string"},{"type":"number"}],"x-busbar-exact-number":{"mantissa_bits":128,"signed":true,"scale":6}},"ratio":{"type":"number"}}}`)
	original := bytes.Clone(input)
	derived, err := deriveSchema(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(input, original) {
		t.Fatal("canonical exact-number schema was modified")
	}
	var document struct {
		Properties map[string]struct {
			GoType string `json:"x-go-type"`
			AnyOf  []struct {
				GoType string `json:"x-go-type"`
				Type   string `json:"type"`
			} `json:"anyOf"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(derived, &document); err != nil {
		t.Fatal(err)
	}
	for property, wanted := range map[string]string{"unsigned": "uint64", "nullable": "uint64", "signed": "int64", "ratio": ""} {
		if got := document.Properties[property].GoType; got != wanted {
			t.Errorf("%s maps to %q, want %q", property, got, wanted)
		}
	}
	branches := document.Properties["count"].AnyOf
	if len(branches) != 2 || branches[0].Type != "string" || branches[0].GoType != "" || branches[1].Type != "number" || branches[1].GoType != "json.Number" {
		t.Fatalf("Count string/numeric representations changed or remained lossy: %+v", branches)
	}
	repeated, err := deriveSchema(derived)
	if err != nil || !bytes.Equal(repeated, derived) {
		t.Fatalf("exact-number mappings are not idempotent: %v", err)
	}
}

func TestDeriveCountSchemaRejectsExactNumberMetadataMismatch(t *testing.T) {
	for _, input := range []string{
		`{"type":"integer","x-busbar-exact-number":{"bits":32,"signed":false}}`,
		`{"type":"integer","x-busbar-exact-number":{"bits":64,"signed":"false"}}`,
		`{"type":"integer","x-busbar-exact-number":{"integer_format":"uint128"}}`,
		`{"type":"number","x-busbar-exact-number":{"integer_format":"uint64"}}`,
		`{"type":"number","x-busbar-exact-number":{"mantissa_bits":128,"signed":true,"scale":7}}`,
		`{"type":"number","x-busbar-exact-number":"invalid"}`,
		`{"type":["integer","string"],"x-busbar-exact-number":{"integer_format":"uint64"}}`,
	} {
		if _, err := deriveSchema([]byte(input)); err == nil {
			t.Errorf("accepted unsupported or mismatched exact-number metadata: %s", input)
		}
	}
}
