package cmd

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rishabh-chatterjee/dashme/internal/config"
	"github.com/rishabh-chatterjee/dashme/internal/github"
	"github.com/rishabh-chatterjee/dashme/internal/ui"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show PR statistics",
	Long:  `Display weekly PR merge and review statistics with charts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		ctx := cmd.Context()
		client := github.NewClient(cfg.Token)

		weekStats, err := client.FetchWeekStats(ctx, cfg.Org, cfg.Username, 0)
		if err != nil {
			return fmt.Errorf("failed to fetch week stats: %w", err)
		}

		members, err := client.FetchTeamMembers(ctx, cfg.Org, cfg.Team)
		if err != nil {
			return fmt.Errorf("failed to fetch team members: %w", err)
		}

		fetchStats := func(ctx context.Context, username string, viewMode ui.ViewMode, offset int) (github.UserStats, error) {
			switch viewMode {
			case ui.QuarterView:
				return client.FetchQuarterStats(ctx, cfg.Org, username, offset)
			case ui.YearView:
				return client.FetchYearStats(ctx, cfg.Org, username, offset)
			default:
				return client.FetchWeekStats(ctx, cfg.Org, username, offset)
			}
		}

		p := tea.NewProgram(ui.NewModel(ctx, weekStats, members, fetchStats), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running UI: %w", err)
		}

		return nil
	},
}

func init() {
}
