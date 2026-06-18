package healthstatus

import (
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const DateLayout = "2006-01-02"

func CleanString(value string) string {
	return strings.TrimSpace(value)
}

func NilIfEmpty(value string) *string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

func ParseDate(value string, field string) (time.Time, error) {
	parsed, err := time.Parse(DateLayout, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must use YYYY-MM-DD", field)
	}
	return parsed, nil
}

func ParseOptionalDate(value *string, field string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := ParseDate(*value, field)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func FormatDate(value pgtype.Date) *string {
	if !value.Valid {
		return nil
	}
	formatted := value.Time.Format(DateLayout)
	return &formatted
}

func FormatRequiredDate(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(DateLayout)
}
