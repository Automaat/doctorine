package illnesses

import "time"

type Illness struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Status      string  `json:"status"`
	DiagnosedOn *string `json:"diagnosed_on"`
	ResolvedOn  *string `json:"resolved_on"`
	Clinician   *string `json:"clinician"`
	Notes       *string `json:"notes"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type CreateParams struct {
	Title       string
	Status      string
	DiagnosedOn *time.Time
	ResolvedOn  *time.Time
	Clinician   *string
	Notes       *string
}
