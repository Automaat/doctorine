package examinations

import "time"

type Examination struct {
	ID           int     `json:"id"`
	Title        string  `json:"title"`
	ExamDate     string  `json:"exam_date"`
	Category     string  `json:"category"`
	Facility     *string `json:"facility"`
	ResultStatus string  `json:"result_status"`
	Summary      *string `json:"summary"`
	Notes        *string `json:"notes"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type CreateParams struct {
	Title        string
	ExamDate     time.Time
	Category     string
	Facility     *string
	ResultStatus string
	Summary      *string
	Notes        *string
}
