package weights

import "testing"

func TestValidateCreate(t *testing.T) {
	notes := " after gym "
	params, detail := validateCreate(createRequest{
		MeasuredOn: "2026-06-20",
		WeightKg:   78.4,
		Notes:      &notes,
	})

	if detail != "" {
		t.Fatal(detail)
	}
	if params.MeasuredOn.Format("2006-01-02") != "2026-06-20" {
		t.Fatalf("MeasuredOn = %v, want 2026-06-20", params.MeasuredOn)
	}
	if params.WeightKg != 78.4 {
		t.Fatalf("WeightKg = %v, want 78.4", params.WeightKg)
	}
	if params.Notes == nil || *params.Notes != "after gym" {
		t.Fatalf("Notes = %v, want after gym", params.Notes)
	}
}

func TestValidateCreateRejectsBadDate(t *testing.T) {
	_, detail := validateCreate(createRequest{
		MeasuredOn: "20-06-2026",
		WeightKg:   78,
	})

	if detail != "Date must use YYYY-MM-DD" {
		t.Fatalf("detail = %q, want Date must use YYYY-MM-DD", detail)
	}
}

func TestValidateCreateRejectsNonPositiveWeight(t *testing.T) {
	_, detail := validateCreate(createRequest{
		MeasuredOn: "2026-06-20",
		WeightKg:   0,
	})

	if detail != "Weight must be greater than 0" {
		t.Fatalf("detail = %q, want Weight must be greater than 0", detail)
	}
}

func TestValidateCreateRejectsImplausibleWeight(t *testing.T) {
	_, detail := validateCreate(createRequest{
		MeasuredOn: "2026-06-20",
		WeightKg:   1500,
	})

	if detail != "Weight must be less than 1000 kg" {
		t.Fatalf("detail = %q, want Weight must be less than 1000 kg", detail)
	}
}
