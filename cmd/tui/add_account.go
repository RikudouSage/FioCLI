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
	"go.chrastecky.dev/fio-client/fioclient/model"
)

type apiKeyFormAction uint8

const (
	apiKeyFormIdle apiKeyFormAction = iota
	apiKeyFormSubmit
	apiKeyFormCancel
)

// apiKeyForm is a reusable secret-input component. It never renders the API
// key itself and delegates account registration to its containing screen.
type apiKeyForm struct {
	input         textinput.Model
	width, height int
	loading       bool
	err           error
}

func newAPIKeyForm() *apiKeyForm {
	input := textinput.New()
	input.Placeholder = "API key"
	input.Prompt = "> "
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '•'
	input.Focus()
	return &apiKeyForm{input: input, width: defaultWidth, height: defaultHeight}
}

func (f *apiKeyForm) Init() tea.Cmd { return textinput.Blink }

func (f *apiKeyForm) Update(msg tea.Msg) (apiKeyFormAction, tea.Cmd) {
	if f.loading {
		return apiKeyFormIdle, nil
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		f.err = nil
		switch keyMsg.String() {
		case "esc":
			return apiKeyFormCancel, nil
		case "enter":
			if strings.TrimSpace(f.input.Value()) == "" {
				f.err = errors.New("API key cannot be empty")
				return apiKeyFormIdle, nil
			}
			return apiKeyFormSubmit, nil
		}
	}
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return apiKeyFormIdle, cmd
}

func (f *apiKeyForm) View() string {
	modalWidth := min(max(f.width-8, 24), 64)
	contentWidth := max(modalWidth-4, 1)
	f.input.Width = max(contentWidth-2, 1)
	lines := []string{titleStyle.Render("Add account"), helpStyle.Render("Enter the API key from Fio internet banking"), "", f.input.View()}
	if f.loading {
		lines = append(lines, "", helpStyle.Render("Adding account and importing transactions…"))
	}
	if f.err != nil {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(outColor).Render(ansi.Truncate(f.err.Error(), contentWidth, "…")))
	}
	lines = append(lines, "", helpStyle.Render("enter add • esc cancel"))
	modal := cardStyle.Width(contentWidth).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(f.width, f.height, lipgloss.Center, lipgloss.Center, modal)
}

type accountRegistrationResult struct {
	accounts []model.Account
	added    model.Account
	err      error
}

type addAccountScreen struct {
	previous        screen
	form            *apiKeyForm
	registerAccount AccountRegistrar
	ctx             context.Context
}

func newAddAccountScreen(previous screen, registerAccount AccountRegistrar, ctx context.Context, width, height int) *addAccountScreen {
	s := &addAccountScreen{previous: previous, form: newAPIKeyForm(), registerAccount: registerAccount, ctx: ctx}
	s.Resize(width, height)
	return s
}

func (s *addAccountScreen) Init() tea.Cmd { return s.form.Init() }

func (s *addAccountScreen) acceptsTextInput() bool { return !s.form.loading }

func (s *addAccountScreen) Update(msg tea.Msg) (screen, tea.Cmd, navigation) {
	if result, ok := msg.(accountRegistrationResult); ok {
		s.form.loading = false
		if result.err != nil {
			s.form.err = result.err
			return s, nil, navigation{}
		}
		if picker, ok := s.previous.(*accountPickerScreen); ok {
			picker.picker.accounts = result.accounts
			for index, account := range result.accounts {
				if account.AccountNumber == result.added.AccountNumber {
					picker.picker.selected = index
					break
				}
			}
		}
		return s.previous, nil, navigation{destination: accountsUpdated, accounts: result.accounts}
	}
	action, cmd := s.form.Update(msg)
	switch action {
	case apiKeyFormCancel:
		return s.previous, nil, navigation{}
	case apiKeyFormSubmit:
		if s.registerAccount == nil {
			s.form.err = errors.New("account registration is unavailable")
			return s, nil, navigation{}
		}
		s.form.loading = true
		apiKey := strings.TrimSpace(s.form.input.Value())
		return s, func() tea.Msg {
			accounts, added, err := s.registerAccount(s.ctx, apiKey)
			if err != nil {
				err = fmt.Errorf("failed adding account: %w", err)
			}
			return accountRegistrationResult{accounts: accounts, added: added, err: err}
		}, navigation{}
	}
	return s, cmd, navigation{}
}

func (s *addAccountScreen) View() string { return s.form.View() }
func (s *addAccountScreen) Resize(width, height int) {
	s.form.width, s.form.height = max(width, 1), max(height, 1)
}
