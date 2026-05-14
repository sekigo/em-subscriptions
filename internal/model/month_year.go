package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// MonthYear represents a calendar month: only month and year matter, the day is
// always normalized to the 1st. The wire format is "MM-YYYY" (e.g. "07-2025"),
// matching the task spec. In the database it is stored as a DATE pointing at
// the first day of the month, which keeps SQL date arithmetic trivial.
type MonthYear struct {
	t time.Time
}

const monthYearLayout = "01-2006"

func NewMonthYear(year int, month time.Month) MonthYear {
	return MonthYear{t: time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)}
}

// FromTime normalizes any time to the first day of its (UTC) month.
func FromTime(t time.Time) MonthYear {
	t = t.UTC()
	return MonthYear{t: time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)}
}

func ParseMonthYear(s string) (MonthYear, error) {
	s = strings.TrimSpace(s)
	t, err := time.ParseInLocation(monthYearLayout, s, time.UTC)
	if err != nil {
		return MonthYear{}, fmt.Errorf("invalid month-year %q: expected MM-YYYY", s)
	}
	return MonthYear{t: t}, nil
}

func (m MonthYear) Time() time.Time { return m.t }
func (m MonthYear) IsZero() bool    { return m.t.IsZero() }
func (m MonthYear) String() string  { return m.t.Format(monthYearLayout) }

// MarshalJSON / UnmarshalJSON — JSON wire format is "MM-YYYY".
func (m MonthYear) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.t.Format(monthYearLayout) + `"`), nil
}

func (m *MonthYear) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		return nil
	}
	v, err := ParseMonthYear(s)
	if err != nil {
		return err
	}
	*m = v
	return nil
}

// Value / Scan — DB representation is DATE (first day of month).
func (m MonthYear) Value() (driver.Value, error) {
	if m.IsZero() {
		return nil, nil
	}
	return m.t, nil
}

func (m *MonthYear) Scan(src any) error {
	if src == nil {
		*m = MonthYear{}
		return nil
	}
	switch v := src.(type) {
	case time.Time:
		*m = FromTime(v)
		return nil
	case string:
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return fmt.Errorf("scan MonthYear from string: %w", err)
		}
		*m = FromTime(t)
		return nil
	default:
		return fmt.Errorf("scan MonthYear: unsupported source type %T", src)
	}
}
