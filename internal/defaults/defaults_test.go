package defaults_test

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/defaults"
)

func TestDefaultConcerns_Length(t *testing.T) {
	concerns := defaults.DefaultConcerns()
	if len(concerns) != 7 {
		t.Errorf("expected 7 concerns, got %d", len(concerns))
	}
}

func TestDefaultConcerns_IDs(t *testing.T) {
	concerns := defaults.DefaultConcerns()
	expectedIDs := []string{
		"open-source",
		"beautiful-product",
		"long-lived",
		"move-fast",
		"compliance",
		"learning-project",
		"well-engineered",
	}
	for i, expected := range expectedIDs {
		if concerns[i].ID != expected {
			t.Errorf("concerns[%d].ID = %q, want %q", i, concerns[i].ID, expected)
		}
	}
}

func TestDefaultConcerns_Names(t *testing.T) {
	concerns := defaults.DefaultConcerns()
	expectedNames := []string{
		"Open Source",
		"Beautiful Product",
		"Long-Lived",
		"Move Fast",
		"Compliance",
		"Learning Project",
		"Well-Engineered",
	}
	for i, expected := range expectedNames {
		if concerns[i].Name != expected {
			t.Errorf("concerns[%d].Name = %q, want %q", i, concerns[i].Name, expected)
		}
	}
}

func TestDefaultConcerns_NonEmpty(t *testing.T) {
	concerns := defaults.DefaultConcerns()
	for _, c := range concerns {
		if c.ID == "" {
			t.Errorf("concern has empty ID")
		}
		if c.Name == "" {
			t.Errorf("concern %q has empty Name", c.ID)
		}
		if c.Description == "" {
			t.Errorf("concern %q has empty Description", c.ID)
		}
	}
}

func TestDefaultConcerns_ReviewDimensions(t *testing.T) {
	concerns := defaults.DefaultConcerns()
	for _, c := range concerns {
		if len(c.ReviewDimensions) == 0 {
			t.Errorf("concern %q has no review dimensions", c.ID)
		}
	}
}

func TestDefaultConcerns_IndependentSlices(t *testing.T) {
	c1 := defaults.DefaultConcerns()
	c2 := defaults.DefaultConcerns()
	if len(c1) != len(c2) {
		t.Errorf("length mismatch: %d vs %d", len(c1), len(c2))
	}
}
