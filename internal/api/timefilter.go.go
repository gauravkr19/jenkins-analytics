package api

import (
	"fmt"
	"time"
)

// DateRange holds start and end dates for filtering
type DateRange struct {
    From time.Time
    To   time.Time
}

// GetDateRange returns the [From, To] span for the given key.
// Supported keys: today, yesterday, this_week, previous_week, this_month, previous_month.
func GetDateRange(key string) (DateRange, error) {
    loc := time.Now().Location()
    now := time.Now().In(loc)
    // zero‐out time of day
    today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

    switch key {
    case "today":
        return DateRange{
            From: today,
            To:   today.AddDate(0, 0, 1).Add(-time.Nanosecond),
        }, nil

    case "yesterday":
        y := today.AddDate(0, 0, -1)
        return DateRange{
            From: y,
            To:   today.Add(-time.Nanosecond),
        }, nil

    case "this_week":
        wd := int(today.Weekday())
        if wd == 0 { wd = 7 }                    // Sunday → 7
        start := today.AddDate(0, 0, -(wd - 1))  // Monday of this week
        end   := start.AddDate(0, 0, 7).Add(-time.Nanosecond)
        return DateRange{From: start, To: end}, nil

    case "previous_week":
        wd := int(today.Weekday())
        if wd == 0 { wd = 7 }
        thisMon := today.AddDate(0, 0, -(wd - 1))
        start   := thisMon.AddDate(0, 0, -7)
        end     := thisMon.Add(-time.Nanosecond)
        return DateRange{From: start, To: end}, nil

    case "this_month":
        y, m := today.Year(), today.Month()
        start := time.Date(y, m, 1, 0, 0, 0, 0, loc)
        end   := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
        return DateRange{From: start, To: end}, nil

    case "previous_month":
        y, m := today.Year(), today.Month()
        // first day of previous month:
        pm := time.Date(y, m, 1, 0, 0, 0, 0, loc).AddDate(0, -1, 0)
        start := time.Date(pm.Year(), pm.Month(), 1, 0, 0, 0, 0, loc)
        end   := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
        return DateRange{From: start, To: end}, nil

    default:
        return DateRange{}, fmt.Errorf("unsupported range key %q", key)
    }
}
