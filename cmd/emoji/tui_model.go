package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/x/term"
)

var (
	colorPrimary = lipgloss.Color("#7D56F4")
	colorAccent  = lipgloss.Color("#FF75B7")
	colorText    = lipgloss.Color("#FAFAFA")
	colorMuted   = lipgloss.Color("#6C6C6C")
	colorSubtle  = lipgloss.Color("#A8A8A8")
	colorSuccess = lipgloss.Color("#04B575")
	colorError   = lipgloss.Color("#FF5F5F")
)

var (
	styleSelected  = lipgloss.NewStyle().Foreground(colorText).Background(colorPrimary).Bold(true)
	styleCursor    = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleNameDim   = lipgloss.NewStyle().Foreground(colorSubtle)
	styleNameMatch = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleHelp      = lipgloss.NewStyle().Foreground(colorMuted)
	styleStatus    = lipgloss.NewStyle().Foreground(colorSuccess)
	styleError     = lipgloss.NewStyle().Foreground(colorError)
	styleEmpty     = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
)

type emoji struct {
	Char     string
	Name     string
	Keywords []string
}

type model struct {
	input           textinput.Model
	emojis          []emoji
	source          searchSource
	results         []result
	cursor          int
	status          string
	quit            bool
	windowSize      tea.WindowSizeMsg
	emojiResultView viewport.Model
}

func initialModel() *model {
	ti := textinput.New()
	ti.Placeholder = "Search emojis..."
	ti.Focus()
	ti.SetWidth(40)

	m := &model{
		input:  ti,
		emojis: emojiData,
		source: buildSource(emojiData),
	}
	m.results = filter(m.emojis, m.source, "")
	return m
}

// chromeHeight is the number of lines the View reserves outside the results
// viewport: input (1) + blank (1) + blank-before-help (1) + help (1) + status (1) + trailing blank (1).
const chromeHeight = 6

func (m *model) Init() tea.Cmd {
	fd := os.Stdout.Fd()
	width, height, _ := term.GetSize(fd)
	m.windowSize = tea.WindowSizeMsg{Width: width, Height: height}
	m.emojiResultView = viewport.New()
	m.resizeViewport()
	return textinput.Blink
}

func (m *model) resizeViewport() {
	w := m.windowSize.Width
	h := m.windowSize.Height - chromeHeight
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	m.emojiResultView.SetWidth(w)
	m.emojiResultView.SetHeight(h)
}

// ensureCursorVisible nudges the viewport's YOffset so the cursor row stays
// inside the visible window as the user navigates with up/down.
func (m *model) ensureCursorVisible() {
	h := m.emojiResultView.Height()
	if h <= 0 {
		return
	}
	off := m.emojiResultView.YOffset()
	if m.cursor < off {
		m.emojiResultView.SetYOffset(m.cursor)
	} else if m.cursor >= off+h {
		m.emojiResultView.SetYOffset(m.cursor - h + 1)
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quit = true
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			m.ensureCursorVisible()
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
			m.ensureCursorVisible()
			return m, nil
		case "enter":
			if len(m.results) == 0 {
				return m, nil
			}
			selected := m.results[m.cursor].Emoji
			if err := clipboard.WriteAll(selected.Char); err != nil {
				m.status = "clipboard error: " + err.Error()
				return m, nil
			}
			m.status = fmt.Sprintf("copied %s (%s)", selected.Char, selected.Name)
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.windowSize = msg
		m.resizeViewport()
	}

	prev := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != prev {
		m.results = filter(m.emojis, m.source, m.input.Value())
		m.cursor = 0
		m.emojiResultView.SetYOffset(0)
	}
	return m, cmd
}

// renderName styles the emoji name with matched runes shown bold/accented.
// matched holds byte offsets into name (sahilm/fuzzy yields byte indexes via
// its for-range scan), which lines up with Go's `for i, r := range name`.
func renderName(name string, matched []int) string {
	if len(matched) == 0 {
		return styleNameDim.Render(name)
	}
	var b strings.Builder
	for i, r := range name {
		s := string(r)

		if slices.Contains(matched, i) {
			b.WriteString(styleNameMatch.Render(s))
		} else {
			b.WriteString(styleNameDim.Render(s))
		}
	}
	return b.String()
}

func (m *model) View() tea.View {
	if m.quit && m.status != "" {
		return tea.NewView(styleStatus.Render(m.status) + "\n")
	}

	var b strings.Builder
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	if len(m.results) == 0 {
		b.WriteString("  " + styleEmpty.Render("no matches") + "\n")
	} else {
		lines := make([]string, len(m.results))
		for i, r := range m.results {
			e := r.Emoji
			if i == m.cursor {
				lines[i] = fmt.Sprintf("%s %s", styleCursor.Render(">"), styleSelected.Render(e.Char+"  "+e.Name))
			} else {
				lines[i] = fmt.Sprintf("  %s  %s", e.Char, renderName(e.Name, r.NameMatch))
			}
		}
		m.emojiResultView.SetContent(strings.Join(lines, "\n"))
		b.WriteString(m.emojiResultView.View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render("enter: copy • ↑/↓: navigate • esc: quit") + "\n")
	if m.status != "" {
		style := styleStatus
		if strings.HasPrefix(m.status, "clipboard error") {
			style = styleError
		}
		b.WriteString(style.Render(m.status) + "\n")
	}
	b.WriteString("\n")

	return tea.NewView(b.String())
}
