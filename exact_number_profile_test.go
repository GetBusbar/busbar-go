package busbar

import (
	"encoding/json"
	"testing"
)

func TestFinalSchemaExactAdjustmentNumericAndStringArms(t *testing.T) {
	for _, token := range []string{
		"9007199254.740993",
		"170141183460469231731687303715884.105727",
		"-170141183460469231731687303715884.105728",
	} {
		t.Run(token, func(t *testing.T) {
			var numeric ExactAdjustReq_Delta_AdditionalProperties
			if err := json.Unmarshal([]byte(token), &numeric); err != nil {
				t.Fatal(err)
			}
			value, err := numeric.AsExactAdjustReqDelta1()
			if err != nil {
				t.Fatal(err)
			}
			number, ok := any(value).(json.Number)
			if !ok || number.String() != token {
				t.Fatalf("numeric Count arm is %T(%v), want exact json.Number(%s)", value, value, token)
			}
			encoded, err := json.Marshal(numeric)
			if err != nil || string(encoded) != token {
				t.Fatalf("legacy numeric arm changed: %s, %v", encoded, err)
			}
			var text ExactAdjustReq_Delta_AdditionalProperties
			if err := json.Unmarshal([]byte(`"`+token+`"`), &text); err != nil {
				t.Fatal(err)
			}
			decimal, err := text.AsExactAdjustReqDelta0()
			if err != nil || decimal != token {
				t.Fatalf("decimal string arm changed: %q, %v", decimal, err)
			}
			encoded, err = json.Marshal(text)
			if err != nil || string(encoded) != `"`+token+`"` {
				t.Fatalf("new string arm changed wire representation: %s, %v", encoded, err)
			}
		})
	}
}

func TestFinalSchemaUnsignedLimitRetainsFullU64(t *testing.T) {
	const token = "18446744073709551615"
	var limit ConfigInputLimit0
	if err := json.Unmarshal([]byte(`{"per":"total","requests":`+token+`}`), &limit); err != nil {
		t.Fatal(err)
	}
	if value, ok := any(limit.Requests).(uint64); !ok || value != ^uint64(0) {
		t.Fatalf("unsigned settings limit is %T(%v), want uint64 max", limit.Requests, limit.Requests)
	}
	encoded, err := json.Marshal(limit)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil || string(fields["requests"]) != token {
		t.Fatalf("unsigned settings limit changed: %s, %v", encoded, err)
	}
}
