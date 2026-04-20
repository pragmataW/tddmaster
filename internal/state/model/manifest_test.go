package model

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// IsVerifierSkipped — nil-safety and field value tests
// ---------------------------------------------------------------------------

func TestIsVerifierSkipped_NilTdd(t *testing.T) {
	m := NosManifest{Tdd: nil}
	if m.IsVerifierSkipped() {
		t.Error("expected false when Tdd is nil, got true")
	}
}

func TestIsVerifierSkipped_TddNotNil_SkipVerifyFalse(t *testing.T) {
	m := NosManifest{Tdd: &Manifest{SkipVerify: false}}
	if m.IsVerifierSkipped() {
		t.Error("expected false when Tdd.SkipVerify is false, got true")
	}
}

func TestIsVerifierSkipped_TddNotNil_SkipVerifyTrue(t *testing.T) {
	m := NosManifest{Tdd: &Manifest{SkipVerify: true}}
	if !m.IsVerifierSkipped() {
		t.Error("expected true when Tdd.SkipVerify is true, got false")
	}
}

// ---------------------------------------------------------------------------
// Table-driven edge cases
// ---------------------------------------------------------------------------

func TestIsVerifierSkipped_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		manifest NosManifest
		want     bool
	}{
		{
			name:     "nil Tdd returns false",
			manifest: NosManifest{Tdd: nil},
			want:     false,
		},
		{
			name:     "Tdd non-nil SkipVerify false returns false",
			manifest: NosManifest{Tdd: &Manifest{SkipVerify: false}},
			want:     false,
		},
		{
			name:     "Tdd non-nil SkipVerify true returns true",
			manifest: NosManifest{Tdd: &Manifest{SkipVerify: true}},
			want:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.manifest.IsVerifierSkipped()
			if got != tc.want {
				t.Errorf("IsVerifierSkipped() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// YAML omitempty: existing manifest without skipVerify defaults to false
// ---------------------------------------------------------------------------

func TestManifest_YAML_MissingSkipVerify_DefaultsFalse(t *testing.T) {
	raw := `
tddMode: true
maxVerificationRetries: 3
`
	var m Manifest
	if err := yaml.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("yaml.Unmarshal failed: %v", err)
	}
	if m.SkipVerify {
		t.Error("expected SkipVerify to default to false when key is absent in YAML")
	}
}

// ---------------------------------------------------------------------------
// YAML roundtrip: SkipVerify=true survives marshal → unmarshal
// ---------------------------------------------------------------------------

func TestManifest_YAML_Roundtrip_SkipVerifyTrue(t *testing.T) {
	original := Manifest{
		TddMode:    true,
		SkipVerify: true,
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}

	var restored Manifest
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatalf("yaml.Unmarshal failed: %v", err)
	}

	if !restored.SkipVerify {
		t.Errorf("SkipVerify not preserved in YAML roundtrip; marshaled bytes: %s", data)
	}
}

// ---------------------------------------------------------------------------
// JSON roundtrip: SkipVerify=true survives marshal → unmarshal
// ---------------------------------------------------------------------------

func TestManifest_JSON_Roundtrip_SkipVerifyTrue(t *testing.T) {
	original := Manifest{
		TddMode:    true,
		SkipVerify: true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var restored Manifest
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if !restored.SkipVerify {
		t.Errorf("SkipVerify not preserved in JSON roundtrip; marshaled bytes: %s", data)
	}
}

// ---------------------------------------------------------------------------
// JSON omitempty: existing manifest without skipVerify defaults to false
// ---------------------------------------------------------------------------

func TestManifest_JSON_MissingSkipVerify_DefaultsFalse(t *testing.T) {
	raw := `{"tddMode":true,"maxVerificationRetries":3}`
	var m Manifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if m.SkipVerify {
		t.Error("expected SkipVerify to default to false when key is absent in JSON")
	}
}

// ---------------------------------------------------------------------------
// Regression: IsTDDEnabled() still works correctly after struct changes
// ---------------------------------------------------------------------------

func TestIsTDDEnabled_Regression_NilTdd(t *testing.T) {
	m := NosManifest{Tdd: nil}
	if m.IsTDDEnabled() {
		t.Error("IsTDDEnabled() regression: expected false when Tdd is nil")
	}
}

func TestIsTDDEnabled_Regression_TddModeTrue(t *testing.T) {
	m := NosManifest{Tdd: &Manifest{TddMode: true}}
	if !m.IsTDDEnabled() {
		t.Error("IsTDDEnabled() regression: expected true when TddMode is true")
	}
}

func TestIsTDDEnabled_Regression_TddModeFalse(t *testing.T) {
	m := NosManifest{Tdd: &Manifest{TddMode: false}}
	if m.IsTDDEnabled() {
		t.Error("IsTDDEnabled() regression: expected false when TddMode is false")
	}
}

// ---------------------------------------------------------------------------
// SkipVerify field struct tag verification via JSON key name
// ---------------------------------------------------------------------------

func TestManifest_JSON_SkipVerify_FieldTagName(t *testing.T) {
	// Verifies the json tag is "skipVerify" (camelCase), not "SkipVerify"
	m := Manifest{SkipVerify: true}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal to map failed: %v", err)
	}

	if _, ok := raw["skipVerify"]; !ok {
		t.Errorf("expected JSON key 'skipVerify' but got keys: %v", raw)
	}
}
