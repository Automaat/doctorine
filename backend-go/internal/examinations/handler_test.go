package examinations

import "testing"

func TestValidateCreateWithResults(t *testing.T) {
	value := 44.0
	minimum := 11.0
	maximum := 34.0
	flag := "H"
	params, detail := validateCreate(createRequest{
		Title:        "Lab report",
		ExamDate:     "2025-10-31",
		Category:     "laboratory",
		ResultStatus: "attention",
		Results: []resultRequest{{
			TestKey:      "ast",
			Name:         "AST",
			ValueText:    stringPtr("44"),
			ValueNumeric: &value,
			Unit:         stringPtr("U/l"),
			ReferenceMin: &minimum,
			ReferenceMax: &maximum,
			Flag:         &flag,
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

func TestValidateResultsRejectsDuplicateKey(t *testing.T) {
	_, detail := validateResults([]resultRequest{
		{TestKey: "ast", Name: "AST", ValueText: stringPtr("44")},
		{TestKey: "ast", Name: "AST duplicate", ValueText: stringPtr("45")},
	})

	if detail != "Result test_key must be unique per examination" {
		t.Fatalf("detail = %q", detail)
	}
}

func TestValidateResultsRejectsInvalidKey(t *testing.T) {
	_, detail := validateResults([]resultRequest{
		{TestKey: "AST", Name: "AST", ValueText: stringPtr("44")},
	})

	if detail != "Result test_key must use lowercase letters, numbers, and underscores" {
		t.Fatalf("detail = %q", detail)
	}
}

func stringPtr(value string) *string {
	return &value
}
