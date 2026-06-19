package examinations

import "testing"

func TestValidateCreateWithResults(t *testing.T) {
	value := 44.0
	minimum := 11.0
	maximum := 34.0
	valueText := "44"
	unit := "U/l"
	params, detail := validateCreate(createRequest{
		Title:        "Lab report",
		ExamDate:     "2025-10-31",
		Category:     "laboratory",
		ResultStatus: "attention",
		Results: []resultRequest{{
			TestKey:      "ast",
			Name:         "AST",
			ValueText:    &valueText,
			ValueNumeric: &value,
			Unit:         &unit,
			ReferenceMin: &minimum,
			ReferenceMax: &maximum,
		}},
	})

	if detail != "" {
		t.Fatal(detail)
	}
	if len(params.Results) != 1 {
		t.Fatalf("results length = %d, want 1", len(params.Results))
	}
	result := params.Results[0]
	if result.TestKey != "ast" || result.DisplayOrder != 1 {
		t.Fatalf("result = %+v", result)
	}
	if result.Flag == nil || *result.Flag != "H" {
		t.Fatalf("flag = %v, want H", result.Flag)
	}
}

func TestValidateResultsComputesLowFlag(t *testing.T) {
	value := 3.0
	minimum := 4.0
	valueText := "3,0"
	params, detail := validateResults([]resultRequest{{
		TestKey:      "leukocyty",
		Name:         "Leukocyty",
		ValueText:    &valueText,
		ValueNumeric: &value,
		ReferenceMin: &minimum,
	}})

	if detail != "" {
		t.Fatal(detail)
	}
	if len(params) != 1 {
		t.Fatalf("results length = %d, want 1", len(params))
	}
	if params[0].Flag == nil || *params[0].Flag != "L" {
		t.Fatalf("flag = %v, want L", params[0].Flag)
	}
}

func TestValidateResultsIgnoresClientFlag(t *testing.T) {
	value := 5.0
	minimum := 4.0
	maximum := 10.0
	clientFlag := "H"
	valueText := "5,0"
	params, detail := validateResults([]resultRequest{{
		TestKey:      "glukoza",
		Name:         "Glukoza",
		ValueText:    &valueText,
		ValueNumeric: &value,
		ReferenceMin: &minimum,
		ReferenceMax: &maximum,
		Flag:         &clientFlag,
	}})

	if detail != "" {
		t.Fatal(detail)
	}
	if len(params) != 1 {
		t.Fatalf("results length = %d, want 1", len(params))
	}
	if params[0].Flag != nil {
		t.Fatalf("flag = %v, want nil", params[0].Flag)
	}
}

func TestValidateResultsRejectsDuplicateKey(t *testing.T) {
	valueText := "44"
	duplicateValueText := "45"
	_, detail := validateResults([]resultRequest{
		{TestKey: "ast", Name: "AST", ValueText: &valueText},
		{TestKey: "ast", Name: "AST duplicate", ValueText: &duplicateValueText},
	})

	if detail != "Result test_key must be unique per examination" {
		t.Fatalf("detail = %q", detail)
	}
}

func TestValidateResultsRejectsInvalidKey(t *testing.T) {
	valueText := "44"
	_, detail := validateResults([]resultRequest{
		{TestKey: "AST", Name: "AST", ValueText: &valueText},
	})

	if detail != "Result test_key must use lowercase letters, numbers, and underscores" {
		t.Fatalf("detail = %q", detail)
	}
}
