package model

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestExecutionOutput_VerifierRequired_FieldExists verifies the field exists on the struct.
// Covers AC-1.
func TestExecutionOutput_VerifierRequired_FieldExists(t *testing.T) {
	var out ExecutionOutput
	rt := reflect.TypeOf(out)
	field, ok := rt.FieldByName("VerifierRequired")
	if !ok {
		t.Fatal("ExecutionOutput does not have a VerifierRequired field")
	}
	if field.Type.Kind() != reflect.Bool {
		t.Fatalf("VerifierRequired should be bool, got %s", field.Type.Kind())
	}
}

// TestExecutionOutput_VerifierRequired_JSONTag verifies the json tag is exactly "verifierRequired".
// Covers AC-1 and AC-4.
func TestExecutionOutput_VerifierRequired_JSONTag(t *testing.T) {
	rt := reflect.TypeOf(ExecutionOutput{})
	field, ok := rt.FieldByName("VerifierRequired")
	if !ok {
		t.Fatal("ExecutionOutput does not have a VerifierRequired field")
	}
	tag := field.Tag.Get("json")
	if tag != "verifierRequired" {
		t.Fatalf("expected json tag to be \"verifierRequired\", got %q", tag)
	}
}

// TestExecutionOutput_VerifierRequired_DefaultFalse verifies the zero value is false.
// Covers AC-3.
func TestExecutionOutput_VerifierRequired_DefaultFalse(t *testing.T) {
	var out ExecutionOutput
	if out.VerifierRequired != false {
		t.Fatal("expected default value of VerifierRequired to be false")
	}
}

// TestExecutionOutput_VerifierRequired_MarshalFalse verifies that false is included in JSON output (no omitempty).
// Covers AC-2.
func TestExecutionOutput_VerifierRequired_MarshalFalse(t *testing.T) {
	out := ExecutionOutput{VerifierRequired: false}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	// Must contain the key even when value is false (no omitempty)
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal of marshaled output failed: %v", err)
	}
	val, exists := m["verifierRequired"]
	if !exists {
		t.Fatal("\"verifierRequired\" key missing from JSON output when value is false — field must not have omitempty")
	}
	if val != false {
		t.Fatalf("expected \"verifierRequired\" to be false, got %v", val)
	}
}

// TestExecutionOutput_VerifierRequired_MarshalTrue verifies that true serializes correctly.
// Covers AC-2.
func TestExecutionOutput_VerifierRequired_MarshalTrue(t *testing.T) {
	out := ExecutionOutput{VerifierRequired: true}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal of marshaled output failed: %v", err)
	}
	val, exists := m["verifierRequired"]
	if !exists {
		t.Fatal("\"verifierRequired\" key missing from JSON output when value is true")
	}
	if val != true {
		t.Fatalf("expected \"verifierRequired\" to be true, got %v", val)
	}
}

// TestExecutionOutput_VerifierRequired_RoundtripTrue verifies marshal→unmarshal preserves true.
// Covers AC-5.
func TestExecutionOutput_VerifierRequired_RoundtripTrue(t *testing.T) {
	original := ExecutionOutput{VerifierRequired: true}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var decoded ExecutionOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if decoded.VerifierRequired != true {
		t.Fatalf("roundtrip failed: expected VerifierRequired=true, got %v", decoded.VerifierRequired)
	}
}

// TestExecutionOutput_VerifierRequired_RoundtripFalse verifies marshal→unmarshal preserves false.
// Covers AC-5.
func TestExecutionOutput_VerifierRequired_RoundtripFalse(t *testing.T) {
	original := ExecutionOutput{VerifierRequired: false}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var decoded ExecutionOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if decoded.VerifierRequired != false {
		t.Fatalf("roundtrip failed: expected VerifierRequired=false, got %v", decoded.VerifierRequired)
	}
}
