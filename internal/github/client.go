package github

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shurcooL/graphql"
	"golang.org/x/oauth2"
)

type Client struct {
	gql *graphql.Client
}

type PeriodStats struct {
	Start       time.Time
	End         time.Time
	PRsMerged   int
	PRsReviewed int
}

type UserStats struct {
	Username string
	Periods  []PeriodStats
}

func NewClient(token string) *Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := graphql.NewClient("https://api.github.com/graphql", httpClient)
	return &Client{gql: client}
}

func (c *Client) FetchWeeklyStats(ctx context.Context, org, username string, numWeeks int) (UserStats, error) {
	startOfThisWeek := startOfWeek(time.Now())
	periods := make([]struct{ start, end time.Time }, numWeeks)
	for i := 0; i < numWeeks; i++ {
		weekEnd := startOfThisWeek.AddDate(0, 0, -7*i)
		weekStart := weekEnd.AddDate(0, 0, -7)
		periods[i] = struct{ start, end time.Time }{weekStart, weekEnd}
	}
	return c.fetchStats(ctx, org, username, periods)
}

func (c *Client) FetchMonthlyStats(ctx context.Context, org, username string, numMonths int) (UserStats, error) {
	now := time.Now()
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	periods := make([]struct{ start, end time.Time }, numMonths)
	for i := 0; i < numMonths; i++ {
		monthEnd := firstOfThisMonth.AddDate(0, -i, 0)
		monthStart := monthEnd.AddDate(0, -1, 0)
		periods[i] = struct{ start, end time.Time }{monthStart, monthEnd}
	}
	return c.fetchStats(ctx, org, username, periods)
}

func (c *Client) fetchStats(ctx context.Context, org, username string, periods []struct{ start, end time.Time }) (UserStats, error) {
	userStats := UserStats{Username: username}
	results := make([]PeriodStats, len(periods))

	var wg sync.WaitGroup
	errChan := make(chan error, len(periods))

	for i, p := range periods {
		wg.Add(1)

		go func(idx int, start, end time.Time) {
			defer wg.Done()

			merged, err := c.countPRsMerged(ctx, org, username, start, end)
			if err != nil {
				errChan <- fmt.Errorf("failed to count merged PRs: %w", err)
				return
			}

			reviewed, err := c.countPRsReviewed(ctx, org, username, start, end)
			if err != nil {
				errChan <- fmt.Errorf("failed to count reviewed PRs: %w", err)
				return
			}

			results[idx] = PeriodStats{
				Start:       start,
				End:         end,
				PRsMerged:   merged,
				PRsReviewed: reviewed,
			}
		}(i, p.start, p.end)
	}

	wg.Wait()
	close(errChan)

	var firstErr error
	for err := range errChan {
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return UserStats{}, firstErr
	}

	for i := len(results) - 1; i >= 0; i-- {
		userStats.Periods = append(userStats.Periods, results[i])
	}

	return userStats, nil
}

func (c *Client) countPRsMerged(ctx context.Context, org, username string, start, end time.Time) (int, error) {
	var query struct {
		Search struct {
			IssueCount int
		} `graphql:"search(query: $query, type: ISSUE, first: 1)"`
	}

	q := fmt.Sprintf("org:%s is:pr is:merged author:%s merged:%s..%s",
		org, username, start.Format("2006-01-02"), end.Format("2006-01-02"))

	variables := map[string]interface{}{
		"query": graphql.String(q),
	}

	err := c.gql.Query(ctx, &query, variables)
	if err != nil {
		return 0, err
	}

	return query.Search.IssueCount, nil
}

func (c *Client) countPRsReviewed(ctx context.Context, org, username string, start, end time.Time) (int, error) {
	var query struct {
		Search struct {
			IssueCount int
		} `graphql:"search(query: $query, type: ISSUE, first: 1)"`
	}

	q := fmt.Sprintf("org:%s is:pr reviewed-by:%s merged:%s..%s",
		org, username, start.Format("2006-01-02"), end.Format("2006-01-02"))

	variables := map[string]interface{}{
		"query": graphql.String(q),
	}

	err := c.gql.Query(ctx, &query, variables)
	if err != nil {
		return 0, err
	}

	return query.Search.IssueCount, nil
}

func startOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, t.Location())
}
