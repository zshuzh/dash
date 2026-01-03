package github

import (
	"context"
	"fmt"
	"time"

	"github.com/rishabh-chatterjee/dashme/internal/timeutil"
	"github.com/shurcooL/graphql"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
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

type Member struct {
	Login string
	Name  string
}

func NewClient(token string) *Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := graphql.NewClient("https://api.github.com/graphql", httpClient)
	return &Client{gql: client}
}

func (c *Client) FetchWeekStats(ctx context.Context, org, username string, offset int) (UserStats, error) {
	now := time.Now()
	refWeek := timeutil.StartOfWeek(now).AddDate(0, 0, 7*offset)
	weekEnd := refWeek.AddDate(0, 0, 7)

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	if weekEnd.After(today) {
		weekEnd = today
	}

	var periods []struct{ start, end time.Time }
	for d := refWeek; d.Before(weekEnd); d = d.AddDate(0, 0, 1) {
		periods = append(periods, struct{ start, end time.Time }{d, d.AddDate(0, 0, 1)})
	}
	return c.fetchStats(ctx, org, username, periods)
}

func (c *Client) FetchQuarterStats(ctx context.Context, org, username string, offset int) (UserStats, error) {
	now := time.Now()
	refQuarter := timeutil.StartOfQuarter(now).AddDate(0, 3*offset, 0)
	quarterEnd := refQuarter.AddDate(0, 3, 0)
	firstMonday := timeutil.StartOfWeek(refQuarter)

	currentWeekStart := timeutil.StartOfWeek(now)
	if quarterEnd.Before(currentWeekStart) || quarterEnd.Equal(currentWeekStart) {
		currentWeekStart = timeutil.StartOfWeek(quarterEnd.AddDate(0, 0, -1))
	}

	var periods []struct{ start, end time.Time }
	for weekStart := firstMonday; !weekStart.After(currentWeekStart) && weekStart.Before(quarterEnd); weekStart = weekStart.AddDate(0, 0, 7) {
		periods = append(periods, struct{ start, end time.Time }{weekStart, weekStart.AddDate(0, 0, 7)})
	}
	return c.fetchStats(ctx, org, username, periods)
}

func (c *Client) FetchYearStats(ctx context.Context, org, username string, offset int) (UserStats, error) {
	now := time.Now()
	refYear := now.Year() + offset
	yearStart := time.Date(refYear, 1, 1, 0, 0, 0, 0, now.Location())
	yearEnd := time.Date(refYear+1, 1, 1, 0, 0, 0, 0, now.Location())

	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if yearEnd.Before(currentMonth) || yearEnd.Equal(currentMonth) {
		currentMonth = yearEnd.AddDate(0, -1, 0)
	}

	var periods []struct{ start, end time.Time }
	for monthStart := yearStart; !monthStart.After(currentMonth) && monthStart.Before(yearEnd); monthStart = monthStart.AddDate(0, 1, 0) {
		periods = append(periods, struct{ start, end time.Time }{monthStart, monthStart.AddDate(0, 1, 0)})
	}
	return c.fetchStats(ctx, org, username, periods)
}

func (c *Client) fetchStats(ctx context.Context, org, username string, periods []struct{ start, end time.Time }) (UserStats, error) {
	userStats := UserStats{Username: username}
	results := make([]PeriodStats, len(periods))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(5)

	for i, p := range periods {
		i, p := i, p
		g.Go(func() error {
			merged, err := c.countPRsMerged(ctx, org, username, p.start, p.end)
			if err != nil {
				return fmt.Errorf("failed to count merged PRs: %w", err)
			}

			reviewed, err := c.countPRsReviewed(ctx, org, username, p.start, p.end)
			if err != nil {
				return fmt.Errorf("failed to count reviewed PRs: %w", err)
			}

			results[i] = PeriodStats{
				Start:       p.start,
				End:         p.end,
				PRsMerged:   merged,
				PRsReviewed: reviewed,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return UserStats{}, err
	}

	userStats.Periods = results
	return userStats, nil
}

func (c *Client) countPRsMerged(ctx context.Context, org, username string, start, endExclusive time.Time) (int, error) {
	var query struct {
		Search struct {
			IssueCount int
		} `graphql:"search(query: $query, type: ISSUE, first: 1)"`
	}

	endInclusive := endExclusive.AddDate(0, 0, -1)
	q := fmt.Sprintf("org:%s is:pr is:merged author:%s merged:%s..%s",
		org, username, start.Format("2006-01-02"), endInclusive.Format("2006-01-02"))

	variables := map[string]interface{}{
		"query": graphql.String(q),
	}

	err := c.gql.Query(ctx, &query, variables)
	if err != nil {
		return 0, err
	}

	return query.Search.IssueCount, nil
}

func (c *Client) countPRsReviewed(ctx context.Context, org, username string, start, endExclusive time.Time) (int, error) {
	var query struct {
		Search struct {
			IssueCount int
		} `graphql:"search(query: $query, type: ISSUE, first: 1)"`
	}

	endInclusive := endExclusive.AddDate(0, 0, -1)
	q := fmt.Sprintf("org:%s is:pr reviewed-by:%s merged:%s..%s",
		org, username, start.Format("2006-01-02"), endInclusive.Format("2006-01-02"))

	variables := map[string]interface{}{
		"query": graphql.String(q),
	}

	err := c.gql.Query(ctx, &query, variables)
	if err != nil {
		return 0, err
	}

	return query.Search.IssueCount, nil
}

func (c *Client) FetchTeamMembers(ctx context.Context, org, team string) ([]Member, error) {
	var query struct {
		Organization struct {
			Team struct {
				Members struct {
					Nodes []struct {
						Login string
						Name  string
					}
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
				} `graphql:"members(first: 100, after: $cursor)"`
			} `graphql:"team(slug: $team)"`
		} `graphql:"organization(login: $org)"`
	}

	var members []Member
	var cursor *string

	for {
		variables := map[string]interface{}{
			"org":    graphql.String(org),
			"team":   graphql.String(team),
			"cursor": (*graphql.String)(cursor),
		}

		if err := c.gql.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("failed to fetch team members: %w", err)
		}

		for _, node := range query.Organization.Team.Members.Nodes {
			members = append(members, Member{Login: node.Login, Name: node.Name})
		}

		if !query.Organization.Team.Members.PageInfo.HasNextPage {
			break
		}
		cursor = &query.Organization.Team.Members.PageInfo.EndCursor
	}

	return members, nil
}
