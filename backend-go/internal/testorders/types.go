package testorders

import "time"

// Order statuses.
const (
	StatusRequested = "requested"
	StatusCompleted = "completed"
	StatusCancelled = "canceled"
)

// Order is a lab test request, typically created by the coach, that the user
// fulfills by getting bloodwork and entering results.
type Order struct {
	ID            int      `json:"id"`
	Source        string   `json:"source"`
	TestKeys      []string `json:"test_keys"`
	Reason        *string  `json:"reason"`
	Status        string   `json:"status"`
	RequestedOn   string   `json:"requested_on"`
	DueOn         *string  `json:"due_on"`
	ExaminationID *int     `json:"examination_id"`
	Notes         *string  `json:"notes"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

// CreateParams is a validated create request.
type CreateParams struct {
	Source   string
	TestKeys []string
	Reason   *string
	DueOn    *time.Time
	Notes    *string
}

// UpdateParams is a validated PATCH request; nil fields are left unchanged.
type UpdateParams struct {
	Status        *string
	ExaminationID *int
	Notes         *string
}
