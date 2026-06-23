package results

// LatestResult is the most recent measurement for a single test_key, flattened
// across examinations so a coach can read current markers in one call.
type LatestResult struct {
	TestKey      string   `json:"test_key"`
	Name         string   `json:"name"`
	ExamDate     string   `json:"exam_date"`
	ValueText    *string  `json:"value_text"`
	ValueNumeric *float64 `json:"value_numeric"`
	ValuePrefix  *string  `json:"value_prefix"`
	Unit         *string  `json:"unit"`
	ReferenceMin *float64 `json:"reference_min"`
	ReferenceMax *float64 `json:"reference_max"`
	Flag         *string  `json:"flag"`
}
