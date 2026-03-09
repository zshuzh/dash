package timeutil

import "time"

type Period struct {
	Start, End time.Time
}

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

func WeekPeriods(offset int) []Period {
	now := time.Now()
	refWeek := StartOfWeek(now).AddDate(0, 0, 7*offset)
	weekEnd := refWeek.AddDate(0, 0, 7)

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	if weekEnd.After(today) {
		weekEnd = today
	}

	var periods []Period
	for d := refWeek; d.Before(weekEnd); d = d.AddDate(0, 0, 1) {
		periods = append(periods, Period{d, d.AddDate(0, 0, 1)})
	}
	return periods
}

func QuarterPeriods(offset int) []Period {
	now := time.Now()
	refQuarter := StartOfQuarter(now).AddDate(0, 3*offset, 0)
	quarterEnd := refQuarter.AddDate(0, 3, 0)
	firstMonday := StartOfWeek(refQuarter)

	currentWeekStart := StartOfWeek(now)
	if quarterEnd.Before(currentWeekStart) || quarterEnd.Equal(currentWeekStart) {
		currentWeekStart = StartOfWeek(quarterEnd.AddDate(0, 0, -1))
	}

	var periods []Period
	for weekStart := firstMonday; !weekStart.After(currentWeekStart) && weekStart.Before(quarterEnd); weekStart = weekStart.AddDate(0, 0, 7) {
		periods = append(periods, Period{weekStart, weekStart.AddDate(0, 0, 7)})
	}
	return periods
}

func YearPeriods(offset int) []Period {
	now := time.Now()
	refYear := now.Year() + offset
	yearStart := time.Date(refYear, 1, 1, 0, 0, 0, 0, now.Location())
	yearEnd := time.Date(refYear+1, 1, 1, 0, 0, 0, 0, now.Location())

	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if yearEnd.Before(currentMonth) || yearEnd.Equal(currentMonth) {
		currentMonth = yearEnd.AddDate(0, -1, 0)
	}

	var periods []Period
	for monthStart := yearStart; !monthStart.After(currentMonth) && monthStart.Before(yearEnd); monthStart = monthStart.AddDate(0, 1, 0) {
		periods = append(periods, Period{monthStart, monthStart.AddDate(0, 1, 0)})
	}
	return periods
}
