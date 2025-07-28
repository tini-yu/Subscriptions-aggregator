package utils

import (
	"time"
	"fmt"
)

func ParseMonthYear(s string) (time.Time, error) {
	const layout = "01-2006"
	t, err := time.Parse(layout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("неверный формат даты %q, ожидается MM-YYYY: %w", s, err)
	}
	return t, nil
}