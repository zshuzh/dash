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

var (
	weekFlag  int
	monthFlag int
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show PR statistics",
	Long:  `Display weekly PR merge and review statistics with charts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if weekFlag <= 0 {
			return fmt.Errorf("--weeks must be a positive integer")
		}
		if monthFlag <= 0 {
			return fmt.Errorf("--months must be a positive integer")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		ctx := context.Background()
		client := github.NewClient(cfg.Token)

		weeklyStats, err := client.FetchWeeklyStats(ctx, cfg.Org, cfg.Username, weekFlag)
		if err != nil {
			return fmt.Errorf("failed to fetch weekly stats: %w", err)
		}

		monthlyStats, err := client.FetchMonthlyStats(ctx, cfg.Org, cfg.Username, monthFlag)
		if err != nil {
			return fmt.Errorf("failed to fetch monthly stats: %w", err)
		}

		p := tea.NewProgram(ui.NewModel(weeklyStats, monthlyStats), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running UI: %w", err)
		}

		return nil
	},
}

func init() {
	statsCmd.Flags().IntVarP(&weekFlag, "weeks", "w", 12, "Number of weeks to show")
	statsCmd.Flags().IntVarP(&monthFlag, "months", "m", 12, "Number of months to show")
}
