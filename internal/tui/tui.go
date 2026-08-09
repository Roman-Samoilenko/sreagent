package tui

import (
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/roman-samoilenko/sreagent/internal/core"
	"github.com/roman-samoilenko/sreagent/pkg/logger"
)

type Model struct {
	orchestrator core.Orchestrator
	textInput    textinput.Model
	messages     []string // каждое сообщение — отдельная строка
	loading      bool
	width        int
	height       int
	scrollOffset int // смещение для скроллинга (0 — последние сообщения)
}

var (
	userStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	botStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
)

func InitialModel(orch core.Orchestrator) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter your question..."
	ti.Focus()
	ti.CharLimit = 2000
	ti.Width = 80

	return Model{
		orchestrator: orch,
		textInput:    ti,
		messages: []string{
			botStyle.Render("Hello! I'm ready to assist you with incident analysis."),
		},
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

type orchestratorResultMsg struct {
	result string
	err    error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 4

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.loading {
				return m, nil
			}
			input := strings.TrimSpace(m.textInput.Value())
			if input == "" {
				return m, nil
			}
			// Добавляем вопрос пользователя
			m.messages = append(m.messages, userStyle.Render("> "+input))
			m.textInput.Reset()
			m.loading = true
			m.scrollOffset = 0
			return m, m.runOrchestrator(input)

		case tea.KeyUp:
			if m.scrollOffset < len(m.messages)-1 {
				m.scrollOffset++
			}
		case tea.KeyDown:
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		}

	case orchestratorResultMsg:
		m.loading = false
		if msg.err != nil {
			m.messages = append(m.messages, errStyle.Render("Error: "+msg.err.Error()))
		} else {
			m.messages = append(m.messages, botStyle.Render(msg.result))
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) runOrchestrator(query string) tea.Cmd {
	return func() tea.Msg {
		// Подавляем посторонний вывод от langchaingo, чтобы не ломать AltScreen
		origStdout := os.Stdout
		origStderr := os.Stderr
		devNull, _ := os.Open(os.DevNull)
		os.Stdout = devNull
		os.Stderr = devNull
		defer func() {
			os.Stdout = origStdout
			os.Stderr = origStderr
			devNull.Close()
		}()

		ctx := context.Background()
		result, err := m.orchestrator.RunTask(ctx, "tui-session", query)
		logger.Debug("Orchestrator result", "result", result, "error", err)
		return orchestratorResultMsg{result: result, err: err}
	}
}

func (m Model) View() string {
	// Вычисляем высоту видимой области для сообщений
	visibleHeight := m.height - 3 // 3 строки: поле ввода + подсказка
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Определяем диапазон сообщений для отображения
	start := len(m.messages) - visibleHeight - m.scrollOffset
	if start < 0 {
		start = 0
	}
	end := len(m.messages)
	if end-start > visibleHeight {
		end = start + visibleHeight
	}
	visibleMsgs := m.messages[start:end]

	var b strings.Builder
	// Заполняем пустые строки сверху, чтобы сообщения прижимались к низу
	emptyLines := visibleHeight - len(visibleMsgs)
	for i := 0; i < emptyLines; i++ {
		b.WriteString("\n")
	}
	for _, msg := range visibleMsgs {
		b.WriteString(msg)
		b.WriteString("\n")
	}

	// Поле ввода и статус
	if m.loading {
		b.WriteString(helpStyle.Render("Processing... Please wait."))
	} else {
		b.WriteString(m.textInput.View())
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑↓ scroll • Enter send • Ctrl+C exit"))

	return b.String()
}

func Run(orch core.Orchestrator) error {
	p := tea.NewProgram(InitialModel(orch), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
