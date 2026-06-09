package spec

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultSettings_MinTestCoverage(t *testing.T) {
	got := DefaultSettings()
	if got.MinTestCoverage != 80 {
		t.Fatalf("DefaultSettings().MinTestCoverage = %d, want 80", got.MinTestCoverage)
	}
}

func TestSettings_MinTestCoverage_JSONRoundTrip(t *testing.T) {
	s := Settings{
		TDDEnabled:               true,
		SkipVerifierEnabled:      false,
		ImportantTaskGateEnabled: false,
		MinTestCoverage:          75,
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if !strings.Contains(string(data), `"minTestCoverage"`) {
		t.Fatalf("JSON missing minTestCoverage key, got: %s", data)
	}

	var got Settings
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.MinTestCoverage != 75 {
		t.Fatalf("round-trip MinTestCoverage = %d, want 75", got.MinTestCoverage)
	}
}

func TestSettings_MinTestCoverage_OmittedUsesDefault(t *testing.T) {
	raw := `{"tddEnabled":true,"skipVerifierEnabled":false,"importantTaskGateEnabled":false}`

	base := DefaultSettings()
	if err := json.Unmarshal([]byte(raw), &base); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if base.MinTestCoverage != 80 {
		t.Fatalf("MinTestCoverage after unmarshal of omitted field = %d, want 80", base.MinTestCoverage)
	}
}

func TestSettings_MinTestCoverage_ExplicitZeroRoundTrips(t *testing.T) {
	s := Settings{
		TDDEnabled:               true,
		SkipVerifierEnabled:      false,
		ImportantTaskGateEnabled: false,
		MinTestCoverage:          0,
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Settings
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.MinTestCoverage != 0 {
		t.Fatalf("round-trip MinTestCoverage = %d, want 0", got.MinTestCoverage)
	}
}

func TestSettings_ClampCoverage(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{-5, 0},
		{0, 0},
		{80, 80},
		{100, 100},
		{150, 100},
	}
	for _, c := range cases {
		s := Settings{MinTestCoverage: c.in}
		s.ClampCoverage()
		if s.MinTestCoverage != c.want {
			t.Errorf("ClampCoverage(%d) = %d, want %d", c.in, s.MinTestCoverage, c.want)
		}
	}
}

func TestExecState_LastModifiedFiles_JSONRoundTrip(t *testing.T) {
	es := ExecState{
		Iteration:         1,
		LastModifiedFiles: []string{"a.go", "b.go"},
	}
	data, err := json.Marshal(es)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if !strings.Contains(string(data), `"lastModifiedFiles"`) {
		t.Fatalf("JSON missing lastModifiedFiles key, got: %s", data)
	}
	var got ExecState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(got.LastModifiedFiles, es.LastModifiedFiles) {
		t.Fatalf("round-trip LastModifiedFiles = %v, want %v", got.LastModifiedFiles, es.LastModifiedFiles)
	}
}

func TestExecState_LastCoverage_JSONRoundTrip(t *testing.T) {
	es := ExecState{
		Iteration:    1,
		LastCoverage: map[string]float64{"task-1": 92, "task-2": 85},
	}

	data, err := json.Marshal(es)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if !strings.Contains(string(data), `"lastCoverage"`) {
		t.Fatalf("JSON missing lastCoverage key, got: %s", data)
	}

	var got ExecState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !reflect.DeepEqual(got.LastCoverage, es.LastCoverage) {
		t.Fatalf("round-trip LastCoverage = %v, want %v", got.LastCoverage, es.LastCoverage)
	}
}

func TestExecState_LastCoverage_OmitEmptyWhenNil(t *testing.T) {
	es := ExecState{Iteration: 1}

	data, err := json.Marshal(es)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if strings.Contains(string(data), `"lastCoverage"`) {
		t.Fatalf("expected lastCoverage to be omitted when nil, got: %s", data)
	}
}

func TestTask_HasNoCoverageField(t *testing.T) {
	task := Task{}
	rt := reflect.TypeOf(task)

	for i := 0; i < rt.NumField(); i++ {
		name := strings.ToLower(rt.Field(i).Name)
		if strings.Contains(name, "coverage") {
			t.Fatalf("Task struct must not have a coverage-related field, found: %s", rt.Field(i).Name)
		}
	}
}
