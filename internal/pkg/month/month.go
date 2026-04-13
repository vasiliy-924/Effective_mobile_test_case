package month

import (
	"fmt"
	"time"
)

const layout = "01-2006"

// Parse parses "MM-YYYY" into the first calendar day of that month (UTC date).
func Parse(s string) (time.Time, error) {
	t, err := time.Parse(layout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected MM-YYYY: %w", err)
	}
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC), nil
}

// Format renders first-of-month as "MM-YYYY".
func Format(t time.Time) string {
	return t.UTC().Format(layout)
}

// MonthCountInclusive returns the number of calendar months from fromMonth to toMonth inclusive.
// Both must be first-of-month dates in UTC.
func MonthCountInclusive(fromMonth, toMonth time.Time) int {
	fy, fm, _ := fromMonth.UTC().Date()
	ty, tm, _ := toMonth.UTC().Date()
	if ty < fy || (ty == fy && tm < fm) {
		return 0
	}
	return (ty-fy)*12 + int(tm-fm) + 1
}
