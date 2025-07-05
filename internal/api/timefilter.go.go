package api

import (
	"fmt"
	"strings"
	"time"
)

func ParseTimeRange(input string) (time.Time, time.Time, error) {
	now := time.Now()

	switch strings.ToLower(input) {
	case "today":
		from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return from, from.Add(24 * time.Hour), nil
	case "yesterday":
		from := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		return from, from.Add(24 * time.Hour), nil
	case "this_week":
		weekday := int(now.Weekday())
		from := now.AddDate(0, 0, -weekday)
		from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
		return from, from.AddDate(0, 0, 7), nil
	case "last_week":
		weekday := int(now.Weekday())
		from := now.AddDate(0, 0, -weekday-7)
		from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
		return from, from.AddDate(0, 0, 7), nil
	case "july", "june", "may":
		monthMap := map[string]time.Month{
			"july": time.July,
			"june": time.June,
			"may":  time.May,
		}
		year := now.Year()
		month := monthMap[input]
		from := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())
		to := from.AddDate(0, 1, 0)
		return from, to, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported time range: %s", input)
	}
}
