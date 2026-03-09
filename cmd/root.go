package cmd

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rishabh-chatterjee/dashme/internal/config"
	"github.com/rishabh-chatterjee/dashme/internal/github"
	"github.com/rishabh-chatterjee/dashme/internal/slack"
	"github.com/rishabh-chatterjee/dashme/internal/ui"
	"github.com/spf13/cobra"
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
		client := github.NewClient(cfg.Token)

		var slackClient *slack.Client
		if cfg.SlackToken != "" {
			slackClient = slack.NewClient(cfg.SlackToken)
		}

		weekStats, err := client.FetchWeekStats(ctx, cfg.Org, cfg.Username, 0)
		if err != nil {
			return fmt.Errorf("failed to fetch week stats: %w", err)
		}

		if slackClient != nil {
			mergeSlackCounts(ctx, slackClient, &weekStats, ui.WeekView, 0)
		}

		fetchStats := func(ctx context.Context, username string, viewMode ui.ViewMode, offset int) (github.UserStats, error) {
			var stats github.UserStats
			var err error
			switch viewMode {
			case ui.QuarterView:
				stats, err = client.FetchQuarterStats(ctx, cfg.Org, username, offset)
			case ui.YearView:
				stats, err = client.FetchYearStats(ctx, cfg.Org, username, offset)
			default:
				stats, err = client.FetchWeekStats(ctx, cfg.Org, username, offset)
			}
			if err != nil {
				return stats, err
			}
			if slackClient != nil {
				mergeSlackCounts(ctx, slackClient, &stats, viewMode, offset)
			}
			return stats, nil
		}

		p := tea.NewProgram(ui.NewModel(ctx, weekStats, fetchStats), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running UI: %w", err)
		}

		return nil
	},
}

func mergeSlackCounts(ctx context.Context, slackClient *slack.Client, stats *github.UserStats, viewMode ui.ViewMode, offset int) {
	var counts []int
	var err error
	switch viewMode {
	case ui.QuarterView:
		counts, err = slackClient.FetchQuarterCounts(ctx, offset)
	case ui.YearView:
		counts, err = slackClient.FetchYearCounts(ctx, offset)
	default:
		counts, err = slackClient.FetchWeekCounts(ctx, offset)
	}
	if err != nil {
		return
	}
	for i := range stats.Periods {
		if i < len(counts) {
			stats.Periods[i].Announcements = counts[i]
		}
	}
}

func Execute() error {
	return rootCmd.Execute()
}
