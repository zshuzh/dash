package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rishabh-chatterjee/dashme/internal/github"
)

type ViewMode int

const (
	WeeklyView ViewMode = iota
	MonthlyView
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	chartStyle = lipgloss.NewStyle().
			Padding(1, 0).
			Margin(1, 0)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	barMergedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	barReviewedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3498db"))

	userStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#3498db")).
			Padding(0, 1).
			Margin(1, 0)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	toggleActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4")).
				Padding(0, 1)

	toggleInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Padding(0, 1)
)

type keyMap struct {
	Quit    key.Binding
	Weekly  key.Binding
	Monthly key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Weekly: key.NewBinding(
		key.WithKeys("w", "W"),
		key.WithHelp("w", "weekly"),
	),
	Monthly: key.NewBinding(
		key.WithKeys("m", "M"),
		key.WithHelp("m", "monthly"),
	),
}

type Model struct {
	weeklyStats  github.UserStats
	monthlyStats github.UserStats
	viewMode     ViewMode
	width        int
	height       int
}

func NewModel(weeklyStats, monthlyStats github.UserStats) Model {
	return Model{
		weeklyStats:  weeklyStats,
		monthlyStats: monthlyStats,
		viewMode:     WeeklyView,
		width:        80,
		height:       24,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Weekly):
			m.viewMode = WeeklyView
		case key.Matches(msg, keys.Monthly):
			m.viewMode = MonthlyView
		}
	}

	return m, nil
}

func (m Model) currentStats() github.UserStats {
	if m.viewMode == MonthlyView {
		return m.monthlyStats
	}
	return m.weeklyStats
}

func (m Model) View() string {
	var b strings.Builder

	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		userStyle.Render(fmt.Sprintf("👤 %s", m.currentStats().Username)),
		"  ",
		m.renderToggle(),
	)
	b.WriteString(header)
	b.WriteString("\n")

	stats := m.currentStats()
	mergedChart := m.renderChart(stats.Periods, "PRs Merged", true)
	reviewedChart := m.renderChart(stats.Periods, "PRs Reviewed", false)

	b.WriteString(mergedChart)
	b.WriteString("\n")
	b.WriteString(reviewedChart)
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("w: weekly  m: monthly  q: quit"))

	return b.String()
}

func (m Model) renderToggle() string {
	var weekly, monthly string

	if m.viewMode == WeeklyView {
		weekly = toggleActiveStyle.Render("Weekly")
		monthly = toggleInactiveStyle.Render("Monthly")
	} else {
		weekly = toggleInactiveStyle.Render("Weekly")
		monthly = toggleActiveStyle.Render("Monthly")
	}

	return weekly + " " + monthly
}

func (m Model) renderChart(periods []github.PeriodStats, title string, isMerged bool) string {
	var b strings.Builder

	if len(periods) == 0 {
		b.WriteString(labelStyle.Render(title))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("No data for this period"))
		b.WriteString("\n")
		return chartStyle.Render(b.String())
	}

	values := make([]int, len(periods))
	total := 0
	maxVal := 0

	for i, p := range periods {
		v := p.PRsReviewed
		if isMerged {
			v = p.PRsMerged
		}
		values[i] = v
		total += v
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	periodLabel := "week"
	if m.viewMode == MonthlyView {
		periodLabel = "month"
	}

	avg := float64(total) / float64(len(values))

	b.WriteString(labelStyle.Render(title))
	b.WriteString(fmt.Sprintf("  Total: %d  |  Avg/%s: %.1f", total, periodLabel, avg))
	b.WriteString("\n\n")

	maxHeight := 8
	barWidth := 3
	colWidth := 5

	spacing := strings.Repeat(" ", colWidth-barWidth)

	for _, v := range values {
		if v == maxVal && v > 0 {
			label := fmt.Sprintf("%d", v)
			padded := padBar(label, colWidth)
			b.WriteString(helpStyle.Render(padded))
		} else {
			b.WriteString(strings.Repeat(" ", colWidth))
		}
	}
	b.WriteString("\n")

	barStyle := barReviewedStyle
	if isMerged {
		barStyle = barMergedStyle
	}

	for row := maxHeight; row >= 1; row-- {
		threshold := float64(row) / float64(maxHeight) * float64(maxVal)
		for _, v := range values {
			switch {
			case float64(v) >= threshold:
				bar := strings.Repeat("█", barWidth)
				b.WriteString(barStyle.Render(bar))
				b.WriteString(spacing)

			case v > 0 && row == int(float64(v)/float64(maxVal)*float64(maxHeight))+1:
				label := fmt.Sprintf("%d", v)
				padded := padBar(label, colWidth)
				b.WriteString(helpStyle.Render(padded))

			case v == 0 && row == 1:
				padded := padBar("0", colWidth)
				b.WriteString(helpStyle.Render(padded))

			default:
				b.WriteString(strings.Repeat(" ", colWidth))
			}
		}
		b.WriteString("\n")
	}

	for _, p := range periods {
		var label string
		if m.viewMode == MonthlyView {
			label = p.Start.Format("Jan")[:1]
		} else {
			label = p.Start.Format("02")
		}
		padded := padBar(label, colWidth)
		b.WriteString(padded)
	}
	b.WriteString("\n")

	return chartStyle.Render(b.String())
}

func padBar(s string, barWidth int) string {
	if len(s) >= barWidth {
		return s[:barWidth]
	}
	return s + strings.Repeat(" ", barWidth-len(s))
}
