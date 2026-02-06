package ui

import (
	"context"
	"fmt"
	"math"
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

const (
	chartMaxHeight = 8
	chartBarWidth  = 3
	chartColWidth  = 5
)

var (
	partialBlocks = []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	blankCol      = strings.Repeat(" ", chartColWidth)
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	chartStyle = lipgloss.NewStyle().
			MarginTop(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	barMergedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	barReviewedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3498db"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	fadedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444"))

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
	Quit       key.Binding
	Weekly     key.Binding
	Quarter    key.Binding
	Yearly     key.Binding
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
	ctx           context.Context
	stats         github.UserStats
	viewMode      ViewMode
	periodOffsets [3]int
	width         int
	height        int

	loading       bool
	err           error
	fetchStats    FetchStatsFunc
}

func (m Model) periodOffset() int {
	return m.periodOffsets[m.viewMode]
}

func NewModel(ctx context.Context, stats github.UserStats, fetchStats FetchStatsFunc) Model {
	return Model{
		ctx:           ctx,
		stats:         stats,
		viewMode:      WeekView,
		periodOffsets: [3]int{0, 0, 0},
		width:         80,
		height:        24,
		fetchStats:    fetchStats,
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
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Weekly):
			if m.viewMode != WeekView {
				m.viewMode = WeekView
				m.loading = true
				m.err = nil
				return m, m.fetchCurrentStatsCmd()
			}
		case key.Matches(msg, keys.Quarter):
			if m.viewMode != QuarterView {
				m.viewMode = QuarterView
				m.loading = true
				m.err = nil
				return m, m.fetchCurrentStatsCmd()
			}
		case key.Matches(msg, keys.Yearly):
			if m.viewMode != YearView {
				m.viewMode = YearView
				m.loading = true
				m.err = nil
				return m, m.fetchCurrentStatsCmd()
			}
		case key.Matches(msg, keys.Next):
			if m.periodOffset() < 0 {
				m.periodOffsets[m.viewMode]++
				m.loading = true
				m.err = nil
				return m, m.fetchCurrentStatsCmd()
			}
		case key.Matches(msg, keys.Prev):
			m.periodOffsets[m.viewMode]--
			m.loading = true
			m.err = nil
			return m, m.fetchCurrentStatsCmd()
		}
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
	return m.fetchStatsCmd(m.stats.Username, m.viewMode, m.periodOffset())
}



func (m Model) View() string {
	var b strings.Builder

	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
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
		b.WriteString(reviewedChart)
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) periodLabel() string {
	now := time.Now()
	switch m.viewMode {
	case WeekView:
		monday := timeutil.StartOfWeek(now).AddDate(0, 0, 7*m.periodOffset())
		return fmt.Sprintf("Week of %s", monday.Format("Jan 2"))
	case QuarterView:
		refQuarter := timeutil.StartOfQuarter(now).AddDate(0, 3*m.periodOffset(), 0)
		q := (refQuarter.Month()-1)/3 + 1
		return fmt.Sprintf("Q%d %d", q, refQuarter.Year())
	default:
		year := now.Year() + m.periodOffset()
		return fmt.Sprintf("%d", year)
	}
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

	allLabels, futureStart := m.getAllLabels()
	numCols := len(allLabels)
	numData := len(periods)

	values := make([]int, numCols)
	total := 0
	maxVal := 0

	for i, p := range periods {
		if i >= numCols {
			break
		}
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

	avg := 0.0
	if numData > 0 {
		avg = float64(total) / float64(numData)
	}

	b.WriteString(labelStyle.Render(title))
	b.WriteString(fmt.Sprintf("  Total: %d  |  Avg/%s: %.1f", total, periodLabel, avg))
	b.WriteString("\n\n")

	maxHeight := chartMaxHeight
	barWidth := chartBarWidth
	colWidth := chartColWidth
	stepsPerRow := len(partialBlocks) - 1
	totalLevels := maxHeight * stepsPerRow

	spacing := strings.Repeat(" ", colWidth-barWidth)

	levels := make([]int, numCols)
	labelRows := make([]int, numCols)
	for i, v := range values {
		if v <= 0 {
			continue
		}
		lvl := int(math.Round(float64(v*totalLevels) / float64(maxVal)))
		if lvl > totalLevels {
			lvl = totalLevels
		}
		levels[i] = lvl

		topRow := (lvl + stepsPerRow - 1) / stepsPerRow
		if topRow >= maxHeight {
			continue
		}
		labelRows[i] = topRow + 1
	}

	for i, v := range values {
		if i >= futureStart || v <= 0 {
			b.WriteString(blankCol)
			continue
		}
		if levels[i] >= totalLevels || labelRows[i] == 0 {
			label := fmt.Sprintf("%d", v)
			padded := padBar(label, colWidth)
			b.WriteString(helpStyle.Render(padded))
		} else {
			b.WriteString(blankCol)
		}
	}
	b.WriteString("\n")

	barStyle := barReviewedStyle
	if isMerged {
		barStyle = barMergedStyle
	}

	for row := maxHeight; row >= 1; row-- {
		rowStart := (row - 1) * stepsPerRow
		rowEnd := row * stepsPerRow

		for i, v := range values {
			if i >= futureStart {
				b.WriteString(blankCol)
				continue
			}

			lvl := levels[i]

			switch {
			case lvl >= rowEnd:
				bar := strings.Repeat("█", barWidth)
				b.WriteString(barStyle.Render(bar))
				b.WriteString(spacing)

			case lvl > rowStart:
				idx := lvl - rowStart
				if idx < 1 {
					idx = 1
				}
				bar := strings.Repeat(partialBlocks[idx], barWidth)
				b.WriteString(barStyle.Render(bar))
				b.WriteString(spacing)

			case v > 0 && row == labelRows[i]:
				label := fmt.Sprintf("%d", v)
				padded := padBar(label, colWidth)
				b.WriteString(helpStyle.Render(padded))

			case v == 0 && row == 1:
				padded := padBar("0", colWidth)
				b.WriteString(helpStyle.Render(padded))

			default:
				b.WriteString(blankCol)
			}
		}
		b.WriteString("\n")
	}

	for i, label := range allLabels {
		padded := padBar(label, colWidth)
		if i >= futureStart {
			b.WriteString(fadedStyle.Render(padded))
		} else {
			b.WriteString(padded)
		}
	}
	b.WriteString("\n")

	return chartStyle.Render(b.String())
}

func (m Model) getAllLabels() (labels []string, futureStart int) {
	now := time.Now()

	switch m.viewMode {
	case WeekView:
		days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
		todayIdx := int(now.Weekday())
		if todayIdx == 0 {
			todayIdx = 7
		}
		refWeek := timeutil.StartOfWeek(now).AddDate(0, 0, 7*m.periodOffset())
		if m.periodOffset() < 0 {
			return days, 7
		}
		if m.periodOffset() == 0 {
			return days, todayIdx
		}
		futureWeekStart := timeutil.StartOfWeek(now).AddDate(0, 0, 7)
		if !refWeek.Before(futureWeekStart) {
			return days, 0
		}
		return days, 7

	case QuarterView:
		refQuarter := timeutil.StartOfQuarter(now).AddDate(0, 3*m.periodOffset(), 0)
		quarterEnd := refQuarter.AddDate(0, 3, 0)
		firstMonday := timeutil.StartOfWeek(refQuarter)
		numWeeks := 0
		for ws := firstMonday; ws.Before(quarterEnd); ws = ws.AddDate(0, 0, 7) {
			numWeeks++
		}
		for i := 1; i <= numWeeks; i++ {
			labels = append(labels, fmt.Sprintf("W%d", i))
		}
		if m.periodOffset() < 0 {
			return labels, numWeeks
		}
		currentWeekStart := timeutil.StartOfWeek(now)
		futureIdx := 0
		for ws := firstMonday; ws.Before(quarterEnd); ws = ws.AddDate(0, 0, 7) {
			if ws.After(currentWeekStart) {
				break
			}
			futureIdx++
		}
		return labels, futureIdx

	default:
		months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
		if m.periodOffset() < 0 {
			return months, 12
		}
		if m.periodOffset() == 0 {
			return months, int(now.Month())
		}
		return months, 0
	}
}

func padBar(s string, barWidth int) string {
	if len(s) >= barWidth {
		return s[:barWidth]
	}
	return s + strings.Repeat(" ", barWidth-len(s))
}
