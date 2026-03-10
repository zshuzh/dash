package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rishabh-chatterjee/dash/internal/timeutil"
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

func (c *Client) FetchCounts(ctx context.Context, periods []timeutil.Period) ([]int, error) {
	results := make([]int, len(periods))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(5)

	for i, p := range periods {
		i, p := i, p
		g.Go(func() error {
			count, err := c.countMessages(ctx, p.Start, p.End)
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

const announcementEmoji = ":meow_megaphone:"

func (c *Client) countMessages(ctx context.Context, start, end time.Time) (int, error) {
	q := fmt.Sprintf("from:me %s after:%s before:%s",
		announcementEmoji,
		start.AddDate(0, 0, -1).Format(time.DateOnly),
		end.Format(time.DateOnly),
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
