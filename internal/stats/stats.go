package stats

import "github.com/rishabh-chatterjee/dashme/internal/timeutil"

type PeriodStats struct {
	Period        timeutil.Period
	PRsMerged     int
	PRsReviewed   int
	Announcements int
}

type UserStats struct {
	Username string
	Periods  []PeriodStats
	HasSlack bool
}
