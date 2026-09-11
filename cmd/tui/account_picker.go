package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

type accountPickerAction uint8

const (
	accountPickerIdle accountPickerAction = iota
	accountPickerCancelled
	accountPickerSelected
	accountPickerRemove
	accountPickerRemoveImmediately
)

// accountPicker is a reusable modal component. It owns selection, keyboard
// handling and presentation, but knows nothing about persistence or navigation.
type accountPicker struct {
	accounts       []model.Account
	current        string
	selected       int
	width, height  int
	loading        bool
	loadingMessage string
	err            error
}

func newAccountPicker(accounts []model.Account, current string) *accountPicker {
	picker := &accountPicker{accounts: accounts, current: current, width: defaultWidth, height: defaultHeight}
	for index, account := range accounts {
		if account.AccountNumber == current {
			picker.selected = index
			break
		}
	}
	return picker
}

func (p *accountPicker) Resize(width, height int) { p.width, p.height = max(width, 1), max(height, 1) }

func (p *accountPicker) Update(msg tea.Msg) accountPickerAction {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok || p.loading {
		return accountPickerIdle
	}
	p.err = nil
	switch keyMsg.String() {
	case "esc", "q", "backspace", "left", "h":
		return accountPickerCancelled
	case "up", "k":
		if p.selected > 0 {
			p.selected--
		}
	case "down", "j":
		if p.selected < len(p.accounts)-1 {
			p.selected++
		}
	case "enter":
		if len(p.accounts) > 0 {
			return accountPickerSelected
		}
	case "d", "delete":
		if len(p.accounts) > 0 {
			return accountPickerRemove
		}
	case "D":
		if len(p.accounts) > 0 {
			return accountPickerRemoveImmediately
		}
	}
	return accountPickerIdle
}

func (p *accountPicker) SelectedAccount() model.Account {
	if len(p.accounts) == 0 {
		return model.Account{}
	}
	return p.accounts[p.selected]
}

func (p *accountPicker) View() string {
	modalWidth := min(max(p.width-8, 24), 64)
	contentWidth := max(modalWidth-4, 1)
	lines := []string{titleStyle.Render("Switch account"), helpStyle.Render("Choose the account to use")}
	if len(p.accounts) == 0 {
		lines = append(lines, "", helpStyle.Render("No accounts configured"))
	}
	for index, account := range p.accounts {
		marker := "  "
		style := lipgloss.NewStyle()
		if index == p.selected {
			marker, style = "› ", style.Foreground(primaryColor).Bold(true)
		}
		name := account.AccountNumber
		if account.BankCode != "" {
			name += "/" + account.BankCode
		}
		suffix := "  " + account.Currency
		if account.AccountNumber == p.current {
			suffix += "  (current)"
		}
		lines = append(lines, style.Render(marker+ansi.Truncate(name+suffix, contentWidth-2, "…")))
	}
	if p.loading {
		lines = append(lines, "", helpStyle.Render(p.loadingMessage))
	}
	if p.err != nil {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(outColor).Render(ansi.Truncate(p.err.Error(), contentWidth, "…")))
	}
	lines = append(lines, "", helpStyle.Render("↑/k ↓/j select • enter switch • d/delete remove"))
	lines = append(lines, helpStyle.Render("esc cancel"))
	modal := cardStyle.Width(contentWidth).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(p.width, p.height, lipgloss.Center, lipgloss.Center, modal)
}

type accountSwitchResult struct {
	account      model.Account
	transactions []model.Transaction
	err          error
}

type accountPickerScreen struct {
	previous      screen
	picker        *accountPicker
	switchAccount AccountSwitcher
	removeAccount AccountRemover
	ctx           context.Context
}

func newAccountPickerScreen(previous screen, accounts []model.Account, current string, switchAccount AccountSwitcher, removeAccount AccountRemover, ctx context.Context, width, height int) *accountPickerScreen {
	s := &accountPickerScreen{previous: previous, picker: newAccountPicker(accounts, current), switchAccount: switchAccount, removeAccount: removeAccount, ctx: ctx}
	s.Resize(width, height)
	return s
}

func (s *accountPickerScreen) Init() tea.Cmd { return nil }

func (s *accountPickerScreen) Update(msg tea.Msg) (screen, tea.Cmd, navigation) {
	if result, ok := msg.(accountRemovalResult); ok {
		s.picker.loading = false
		s.picker.loadingMessage = ""
		if result.err != nil {
			s.picker.err = result.err
			return s, nil, navigation{}
		}
		return s, nil, removalNavigation(result.state)
	}
	if result, ok := msg.(accountSwitchResult); ok {
		s.picker.loading = false
		s.picker.loadingMessage = ""
		if result.err != nil {
			s.picker.err = result.err
			return s, nil, navigation{}
		}
		return s, nil, navigation{destination: selectAccount, account: result.account, transactions: result.transactions}
	}
	switch s.picker.Update(msg) {
	case accountPickerCancelled:
		return s.previous, nil, navigation{}
	case accountPickerSelected:
		if s.switchAccount == nil {
			s.picker.err = errors.New("account switching is unavailable")
			return s, nil, navigation{}
		}
		s.picker.loading = true
		s.picker.loadingMessage = "Switching account…"
		accountNumber := s.picker.SelectedAccount().AccountNumber
		return s, func() tea.Msg {
			account, transactions, err := s.switchAccount(s.ctx, accountNumber)
			if err != nil {
				err = fmt.Errorf("failed switching account: %w", err)
			}
			return accountSwitchResult{account: account, transactions: transactions, err: err}
		}, navigation{}
	case accountPickerRemove:
		return s, nil, navigation{destination: showRemoveAccountConfirmation, account: s.picker.SelectedAccount()}
	case accountPickerRemoveImmediately:
		if s.removeAccount == nil {
			s.picker.err = errors.New("account removal is unavailable")
			return s, nil, navigation{}
		}
		s.picker.loading = true
		s.picker.loadingMessage = "Removing account…"
		return s, removeAccountCmd(s.ctx, s.removeAccount, s.picker.SelectedAccount().AccountNumber), navigation{}
	}
	return s, nil, navigation{}
}

func (s *accountPickerScreen) View() string             { return s.picker.View() }
func (s *accountPickerScreen) Resize(width, height int) { s.picker.Resize(width, height) }
