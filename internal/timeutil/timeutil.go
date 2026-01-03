package timeutil

import "time"

func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, t.Location())
}

func StartOfQuarter(t time.Time) time.Time {
	month := t.Month()
	var quarterStartMonth time.Month
	switch {
	case month <= 3:
		quarterStartMonth = 1
	case month <= 6:
		quarterStartMonth = 4
	case month <= 9:
		quarterStartMonth = 7
	default:
		quarterStartMonth = 10
	}
	return time.Date(t.Year(), quarterStartMonth, 1, 0, 0, 0, 0, t.Location())
}
