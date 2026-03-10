package cmd

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rishabh-chatterjee/dashme/internal/config"
	"github.com/rishabh-chatterjee/dashme/internal/github"
	"github.com/rishabh-chatterjee/dashme/internal/slack"
	"github.com/rishabh-chatterjee/dashme/internal/stats"
	"github.com/rishabh-chatterjee/dashme/internal/timeutil"
	"github.com/rishabh-chatterjee/dashme/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var rootCmd = &cobra.Command{
	Use:                "dashme",
	Short:              "A CLI dashboard for GitHub PR statistics",
	Long:               `dashme shows weekly summaries of PRs merged and reviewed for you and your colleagues.`,
	DisableAutoGenTag:  true,
	CompletionOptions:  cobra.CompletionOptions{DisableDefaultCmd: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		ctx := cmd.Context()
		ghClient := github.NewClient(cfg.Token)

		var slackClient *slack.Client
		if cfg.SlackToken != "" {
			slackClient = slack.NewClient(cfg.SlackToken)
		}

		weekStats, err := fetchAllStats(ctx, ghClient, slackClient, cfg.Org, cfg.Username, timeutil.WeekPeriods(0))
		if err != nil {
			return fmt.Errorf("failed to fetch week stats: %w", err)
		}

		fetchStats := func(ctx context.Context, username string, viewMode ui.ViewMode, offset int) (stats.UserStats, error) {
			return fetchAllStats(ctx, ghClient, slackClient, cfg.Org, username, periodsForView(viewMode, offset))
		}

		p := tea.NewProgram(ui.NewModel(ctx, weekStats, fetchStats), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running UI: %w", err)
		}

		return nil
	},
}

func periodsForView(viewMode ui.ViewMode, offset int) []timeutil.Period {
	switch viewMode {
	case ui.QuarterView:
		return timeutil.QuarterPeriods(offset)
	case ui.YearView:
		return timeutil.YearPeriods(offset)
	default:
		return timeutil.WeekPeriods(offset)
	}
}

func fetchAllStats(ctx context.Context, ghClient *github.Client, slackClient *slack.Client, org, username string, periods []timeutil.Period) (stats.UserStats, error) {
	var userStats stats.UserStats
	var slackCounts []int

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		userStats, err = ghClient.FetchStats(ctx, org, username, periods)
		return err
	})

	if slackClient != nil {
		g.Go(func() error {
			var err error
			slackCounts, err = slackClient.FetchCounts(ctx, periods)
			if err != nil {
				slackCounts = nil
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return stats.UserStats{}, err
	}

	userStats.HasSlack = slackClient != nil
	for i := range userStats.Periods {
		if i < len(slackCounts) {
			userStats.Periods[i].Announcements = slackCounts[i]
		}
	}

	return userStats, nil
}

func Execute() error {
	return rootCmd.Execute()
}
