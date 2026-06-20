package supplements

type Supplement struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Value     string  `json:"value"`
	Frequency string  `json:"frequency"`
	Notes     *string `json:"notes"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type CreateParams struct {
	Name      string
	Value     string
	Frequency string
	Notes     *string
}
