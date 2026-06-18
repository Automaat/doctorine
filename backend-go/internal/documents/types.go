package documents

import "time"

type Document struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	DocumentType     string  `json:"document_type"`
	IssuedAt         *string `json:"issued_at"`
	OriginalFilename string  `json:"original_filename"`
	StorageName      string  `json:"-"`
	ContentType      string  `json:"content_type"`
	SizeBytes        int64   `json:"size_bytes"`
	SHA256Hex        string  `json:"sha256_hex"`
	Notes            *string `json:"notes"`
	IllnessID        *int    `json:"illness_id"`
	IllnessTitle     *string `json:"illness_title"`
	ExaminationID    *int    `json:"examination_id"`
	ExaminationTitle *string `json:"examination_title"`
	CreatedAt        string  `json:"created_at"`
}

type CreateParams struct {
	Title            string
	DocumentType     string
	IssuedAt         *time.Time
	OriginalFilename string
	StorageName      string
	ContentType      string
	SizeBytes        int64
	SHA256Hex        string
	Notes            *string
	IllnessID        *int
	ExaminationID    *int
}
