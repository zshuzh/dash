package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rishabh-chatterjee/dashme/internal/timeutil"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) FetchWeekCounts(ctx context.Context, offset int) ([]int, error) {
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
	return c.fetchCounts(ctx, periods)
}

func (c *Client) FetchQuarterCounts(ctx context.Context, offset int) ([]int, error) {
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
	return c.fetchCounts(ctx, periods)
}

func (c *Client) FetchYearCounts(ctx context.Context, offset int) ([]int, error) {
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
	return c.fetchCounts(ctx, periods)
}

func (c *Client) fetchCounts(ctx context.Context, periods []struct{ start, end time.Time }) ([]int, error) {
	results := make([]int, len(periods))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(5)

	for i, p := range periods {
		i, p := i, p
		g.Go(func() error {
			count, err := c.countMessages(ctx, p.start, p.end)
			if err != nil {
				return err
			}
			results[i] = count
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

type searchResponse struct {
	OK       bool   `json:"ok"`
	Error    string `json:"error"`
	Messages struct {
		Total int `json:"total"`
	} `json:"messages"`
}

func (c *Client) countMessages(ctx context.Context, start, end time.Time) (int, error) {
	q := fmt.Sprintf("from:me :meow_megaphone: after:%s before:%s",
		start.AddDate(0, 0, -1).Format("2006-01-02"),
		end.Format("2006-01-02"),
	)

	u := "https://slack.com/api/search.messages?" + url.Values{
		"query": {q},
		"count": {"1"},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decoding slack response: %w", err)
	}
	if !result.OK {
		return 0, fmt.Errorf("slack API error: %s", result.Error)
	}
	return result.Messages.Total, nil
}
