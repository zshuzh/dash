package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rishabh-chatterjee/dashme/internal/github"
	"github.com/rishabh-chatterjee/dashme/internal/timeutil"
)

type ViewMode int

const (
	WeekView ViewMode = iota
	QuarterView
	YearView
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

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4")).
				Padding(0, 1)

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Padding(0, 1)
	)

type keyMap struct {
	Quit       key.Binding
	Weekly     key.Binding
	Quarter    key.Binding
	Yearly     key.Binding
	UserSelect key.Binding
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Escape     key.Binding
	Next       key.Binding
	Prev       key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Weekly: key.NewBinding(
		key.WithKeys("w", "W"),
		key.WithHelp("w", "week"),
	),
	Quarter: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quarter"),
	),
	Yearly: key.NewBinding(
		key.WithKeys("y", "Y"),
		key.WithHelp("y", "year"),
	),
	UserSelect: key.NewBinding(
		key.WithKeys("u", "U"),
		key.WithHelp("u", "user"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k", "ctrl+p"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j", "ctrl+n"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
	),
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev"),
	),
}

type statsMsg struct {
	stats github.UserStats
}

type statsErrMsg struct {
	err error
}

type FetchStatsFunc func(ctx context.Context, username string, viewMode ViewMode, offset int) (stats github.UserStats, err error)

type Model struct {
	ctx          context.Context
	stats        github.UserStats
	viewMode     ViewMode
	periodOffset int
	width        int
	height       int

	members       []github.Member
	selectingUser bool
	userCursor    int
	loading       bool
	err           error
	fetchStats    FetchStatsFunc
}

func NewModel(ctx context.Context, stats github.UserStats, members []github.Member, fetchStats FetchStatsFunc) Model {
	return Model{
		ctx:          ctx,
		stats:        stats,
		viewMode:     WeekView,
		periodOffset: 0,
		width:        80,
		height:       24,
		members:      members,
		fetchStats:   fetchStats,
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

	case statsMsg:
		m.loading = false
		m.err = nil
		m.stats = msg.stats
		return m, nil

	case statsErrMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		if m.selectingUser {
			return m.handleUserSelectKeys(msg)
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Weekly):
			if m.viewMode != WeekView {
				m.viewMode = WeekView
				m.periodOffset = 0
				m.loading = true
				m.err = nil
				return m, m.fetchCurrentStatsCmd()
			}
		case key.Matches(msg, keys.Quarter):
			if m.viewMode != QuarterView {
				m.viewMode = QuarterView
				m.periodOffset = 0
				m.loading = true
				m.err = nil
				return m, m.fetchCurrentStatsCmd()
			}
		case key.Matches(msg, keys.Yearly):
			if m.viewMode != YearView {
				m.viewMode = YearView
				m.periodOffset = 0
				m.loading = true
				m.err = nil
				return m, m.fetchCurrentStatsCmd()
			}
		case key.Matches(msg, keys.UserSelect):
			if len(m.members) > 0 {
				m.selectingUser = true
				m.userCursor = 0
			}
		case key.Matches(msg, keys.Next):
			if m.periodOffset < 0 {
				m.periodOffset++
				m.loading = true
				m.err = nil
				return m, m.fetchCurrentStatsCmd()
			}
		case key.Matches(msg, keys.Prev):
			m.periodOffset--
			m.loading = true
			m.err = nil
			return m, m.fetchCurrentStatsCmd()
		}
	}

	return m, nil
}

func (m Model) handleUserSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.selectingUser = false
		return m, nil

	case key.Matches(msg, keys.Up):
		if m.userCursor > 0 {
			m.userCursor--
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		if m.userCursor < len(m.members)-1 {
			m.userCursor++
		}
		return m, nil

	case key.Matches(msg, keys.Enter):
		selectedUser := m.members[m.userCursor].Login
		m.selectingUser = false
		m.loading = true
		m.err = nil
		m.periodOffset = 0
		return m, m.fetchStatsCmd(selectedUser, m.viewMode, 0)

	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) fetchStatsCmd(username string, viewMode ViewMode, offset int) tea.Cmd {
	return func() tea.Msg {
		ctx := m.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		stats, err := m.fetchStats(ctx, username, viewMode, offset)
		if err != nil {
			return statsErrMsg{err: err}
		}
		return statsMsg{stats: stats}
	}
}

func (m Model) fetchCurrentStatsCmd() tea.Cmd {
	return m.fetchStatsCmd(m.stats.Username, m.viewMode, m.periodOffset)
}



func (m Model) View() string {
	if m.selectingUser {
		return m.renderUserSelect()
	}

	var b strings.Builder

	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		userStyle.Render(fmt.Sprintf("@%s", m.stats.Username)),
		"  ",
		m.renderToggle(),
		"  ",
		titleStyle.Render(m.periodLabel()),
	)
	b.WriteString(header)
	b.WriteString("\n")

	if m.loading {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Loading..."))
		b.WriteString("\n")
	} else if m.err != nil {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	} else {
		mergedChart := m.renderChart(m.stats.Periods, "PRs Merged", true)
		reviewedChart := m.renderChart(m.stats.Periods, "PRs Reviewed", false)

		b.WriteString(mergedChart)
		b.WriteString("\n")
		b.WriteString(reviewedChart)
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("w: week  q: quarter  y: year  tab/shift+tab: navigate  u: user  ctrl+c: quit"))

	return b.String()
}

func (m Model) periodLabel() string {
	now := time.Now()
	switch m.viewMode {
	case WeekView:
		monday := timeutil.StartOfWeek(now).AddDate(0, 0, 7*m.periodOffset)
		return fmt.Sprintf("Week of %s", monday.Format("Jan 2"))
	case QuarterView:
		refQuarter := timeutil.StartOfQuarter(now).AddDate(0, 3*m.periodOffset, 0)
		q := (refQuarter.Month()-1)/3 + 1
		return fmt.Sprintf("Q%d %d", q, refQuarter.Year())
	default:
		year := now.Year() + m.periodOffset
		return fmt.Sprintf("%d", year)
	}
}

func (m Model) renderUserSelect() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select User"))
	b.WriteString("\n\n")

	for i, member := range m.members {
		display := "@" + member.Login
		if member.Name != "" {
			display = fmt.Sprintf("@%s (%s)", member.Login, member.Name)
		}
		if i == m.userCursor {
			b.WriteString(selectedItemStyle.Render("> " + display))
		} else {
			b.WriteString(itemStyle.Render("  " + display))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/k ↓/j: navigate  enter: select  esc: cancel  ctrl+c: quit"))

	return b.String()
}

func (m Model) renderToggle() string {
	week := toggleInactiveStyle.Render("Week")
	quarter := toggleInactiveStyle.Render("Quarter")
	year := toggleInactiveStyle.Render("Year")

	switch m.viewMode {
	case WeekView:
		week = toggleActiveStyle.Render("Week")
	case QuarterView:
		quarter = toggleActiveStyle.Render("Quarter")
	case YearView:
		year = toggleActiveStyle.Render("Year")
	}

	return week + " " + quarter + " " + year
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

	var periodLabel string
	switch m.viewMode {
	case WeekView:
		periodLabel = "day"
	case QuarterView:
		periodLabel = "week"
	default:
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

	for i, p := range periods {
		var label string
		switch m.viewMode {
		case WeekView:
			label = p.Start.Format("Mon")[:2]
		case QuarterView:
			label = fmt.Sprintf("W%d", i+1)
		default:
			label = p.Start.Format("Jan")[:3]
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
