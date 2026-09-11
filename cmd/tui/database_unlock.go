package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type databaseUnlockResult struct {
	state AccountState
	err   error
}

type databaseUnlockScreen struct {
	input   textinput.Model
	unlock  DatabaseUnlocker
	ctx     context.Context
	width   int
	height  int
	loading bool
	err     error
}

func newDatabaseUnlockScreen(unlock DatabaseUnlocker, ctx context.Context, width, height int) *databaseUnlockScreen {
	input := textinput.New()
	input.Placeholder = "Database password"
	input.Prompt = "> "
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '•'
	input.Focus()
	return &databaseUnlockScreen{input: input, unlock: unlock, ctx: ctx, width: width, height: height}
}

func (s *databaseUnlockScreen) Init() tea.Cmd { return textinput.Blink }

func (s *databaseUnlockScreen) acceptsTextInput() bool { return !s.loading }

func (s *databaseUnlockScreen) Update(msg tea.Msg) (screen, tea.Cmd, navigation) {
	if result, ok := msg.(databaseUnlockResult); ok {
		s.loading = false
		if result.err != nil {
			s.err = result.err
			return s, nil, navigation{}
		}
		return s, nil, navigation{destination: databaseUnlocked, account: result.state.Account, transactions: result.state.Transactions, accounts: result.state.Accounts}
	}
	if s.loading {
		return s, nil, navigation{}
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		s.err = nil
		switch key.String() {
		case "esc":
			return s, nil, navigation{destination: exitApplication}
		case "enter":
			password := s.input.Value()
			if password == "" {
				s.err = errors.New("database password cannot be empty")
				return s, nil, navigation{}
			}
			if s.unlock == nil {
				s.err = errors.New("database unlock is unavailable")
				return s, nil, navigation{}
			}
			s.loading = true
			return s, func() tea.Msg {
				state, err := s.unlock(s.ctx, password)
				if err != nil {
					err = fmt.Errorf("failed unlocking database: %w", err)
				}
				return databaseUnlockResult{state: state, err: err}
			}, navigation{}
		}
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd, navigation{}
}

func (s *databaseUnlockScreen) View() string {
	modalWidth := min(max(s.width-8, 24), 64)
	contentWidth := max(modalWidth-4, 1)
	s.input.Width = max(contentWidth-2, 1)
	lines := []string{titleStyle.Render("Unlock database"), helpStyle.Render("Enter the password for your local Fio database"), "", s.input.View()}
	if s.loading {
		lines = append(lines, "", helpStyle.Render("Unlocking database…"))
	}
	if s.err != nil {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(outColor).Render(ansi.Truncate(s.err.Error(), contentWidth, "…")))
	}
	lines = append(lines, "", helpStyle.Render("enter unlock • esc quit"))
	modal := cardStyle.Width(contentWidth).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, modal)
}

func (s *databaseUnlockScreen) Resize(width, height int) {
	s.width, s.height = max(width, 1), max(height, 1)
}
