package weights

import "time"

type Entry struct {
	ID         int     `json:"id"`
	MeasuredOn string  `json:"measured_on"`
	WeightKg   float64 `json:"weight_kg"`
	Notes      *string `json:"notes"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type CreateParams struct {
	MeasuredOn time.Time
	WeightKg   float64
	Notes      *string
}
