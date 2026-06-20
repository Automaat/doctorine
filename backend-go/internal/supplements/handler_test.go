package supplements

import "testing"

func TestValidateCreate(t *testing.T) {
	notes := "with breakfast"
	params, detail := validateCreate(createRequest{
		Name:      " Omega 3 ",
		Value:     " 1000mg ",
		Frequency: " daily ",
		Notes:     &notes,
	})

	if detail != "" {
		t.Fatal(detail)
	}
	if params.Name != "Omega 3" {
		t.Fatalf("Name = %q, want Omega 3", params.Name)
	}
	if params.Value != "1000mg" {
		t.Fatalf("Value = %q, want 1000mg", params.Value)
	}
	if params.Frequency != "daily" {
		t.Fatalf("Frequency = %q, want daily", params.Frequency)
	}
	if params.Notes == nil || *params.Notes != "with breakfast" {
		t.Fatalf("Notes = %v, want with breakfast", params.Notes)
	}
}

func TestValidateCreateRequiresFrequency(t *testing.T) {
	_, detail := validateCreate(createRequest{
		Name:  "Omega 3",
		Value: "1000mg",
	})

	if detail != "Frequency is required" {
		t.Fatalf("detail = %q, want Frequency is required", detail)
	}
}
