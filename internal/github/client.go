package github

import (
	"context"
	"fmt"
	"time"

	"github.com/rishabh-chatterjee/dashme/internal/stats"
	"github.com/rishabh-chatterjee/dashme/internal/timeutil"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	gql *githubv4.Client
}

func NewClient(token string) *Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)
	return &Client{gql: client}
}

func (c *Client) FetchStats(ctx context.Context, org, username string, periods []timeutil.Period) (stats.UserStats, error) {
	userStats := stats.UserStats{Username: username}
	results := make([]stats.PeriodStats, len(periods))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(5)

	for i, p := range periods {
		i, p := i, p
		g.Go(func() error {
			merged, err := c.countPRsMerged(ctx, org, username, p.Start, p.End)
			if err != nil {
				return fmt.Errorf("failed to count merged PRs: %w", err)
			}
			results[i].Period = p
			results[i].PRsMerged = merged
			return nil
		})
		g.Go(func() error {
			reviewed, err := c.countPRsReviewed(ctx, org, username, p.Start, p.End)
			if err != nil {
				return fmt.Errorf("failed to count reviewed PRs: %w", err)
			}
			results[i].Period = p
			results[i].PRsReviewed = reviewed
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return stats.UserStats{}, err
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
		org, username, start.Format(time.DateOnly), endInclusive.Format(time.DateOnly))

	variables := map[string]interface{}{
		"query": githubv4.String(q),
	}

	err := c.gql.Query(ctx, &query, variables)
	if err != nil {
		return 0, err
	}

	return query.Search.IssueCount, nil
}

func (c *Client) countPRsReviewed(ctx context.Context, org, username string, start, endExclusive time.Time) (int, error) {
	var query struct {
		User struct {
			ContributionsCollection struct {
				TotalPullRequestReviewContributions int
			} `graphql:"contributionsCollection(from: $from, to: $to)"`
		} `graphql:"user(login: $username)"`
	}

	variables := map[string]interface{}{
		"username": githubv4.String(username),
		"from":     githubv4.DateTime{Time: start.UTC()},
		"to":       githubv4.DateTime{Time: endExclusive.UTC()},
	}

	err := c.gql.Query(ctx, &query, variables)
	if err != nil {
		return 0, err
	}

	return query.User.ContributionsCollection.TotalPullRequestReviewContributions, nil
}
