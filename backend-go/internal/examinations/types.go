package examinations

import "time"

type Examination struct {
	ID           int      `json:"id"`
	Title        string   `json:"title"`
	ExamDate     string   `json:"exam_date"`
	Category     string   `json:"category"`
	Facility     *string  `json:"facility"`
	ResultStatus string   `json:"result_status"`
	Summary      *string  `json:"summary"`
	Notes        *string  `json:"notes"`
	Results      []Result `json:"results"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

type Result struct {
	ID            int      `json:"id"`
	ExaminationID int      `json:"examination_id"`
	TestKey       string   `json:"test_key"`
	Name          string   `json:"name"`
	ValueText     *string  `json:"value_text"`
	ValueNumeric  *float64 `json:"value_numeric"`
	ValuePrefix   *string  `json:"value_prefix"`
	Unit          *string  `json:"unit"`
	ReferenceMin  *float64 `json:"reference_min"`
	ReferenceMax  *float64 `json:"reference_max"`
	Flag          *string  `json:"flag"`
	DisplayOrder  int      `json:"display_order"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type CreateParams struct {
	Title        string
	ExamDate     time.Time
	Category     string
	Facility     *string
	ResultStatus string
	Summary      *string
	Notes        *string
	Results      []ResultParams
}

type ResultParams struct {
	TestKey      string
	Name         string
	ValueText    *string
	ValueNumeric *float64
	ValuePrefix  *string
	Unit         *string
	ReferenceMin *float64
	ReferenceMax *float64
	Flag         *string
	DisplayOrder int
}
