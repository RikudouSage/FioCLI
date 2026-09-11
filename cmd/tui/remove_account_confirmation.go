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

type confirmationAction uint8

const (
	confirmationIdle confirmationAction = iota
	confirmationAccepted
	confirmationCancelled
)

// confirmation is a reusable destructive-action confirmation component.
type confirmation struct {
	title, message string
	width, height  int
	loading        bool
	err            error
}

func (c *confirmation) Update(msg tea.Msg) confirmationAction {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok || c.loading {
		return confirmationIdle
	}
	c.err = nil
	switch keyMsg.String() {
	case "y", "enter":
		return confirmationAccepted
	case "n", "esc", "q", "backspace", "left", "h":
		return confirmationCancelled
	}
	return confirmationIdle
}

func (c *confirmation) View() string {
	modalWidth := min(max(c.width-8, 24), 64)
	contentWidth := max(modalWidth-4, 1)
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(outColor).Render(c.title),
		"",
		ansi.Truncate(c.message, contentWidth, "…"),
		"",
		helpStyle.Render("y/enter confirm • n/esc cancel"),
		helpStyle.Render("Tip: use Shift+D in the account picker to skip confirmation"),
	}
	if c.loading {
		lines = append(lines, "", helpStyle.Render("Removing account…"))
	}
	if c.err != nil {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(outColor).Render(ansi.Truncate(c.err.Error(), contentWidth, "…")))
	}
	modal := cardStyle.BorderForeground(outColor).Width(contentWidth).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(c.width, c.height, lipgloss.Center, lipgloss.Center, modal)
}

type accountRemovalResult struct {
	state AccountState
	err   error
}

func removeAccountCmd(ctx context.Context, remover AccountRemover, accountNumber string) tea.Cmd {
	return func() tea.Msg {
		state, err := remover(ctx, accountNumber)
		if err != nil {
			err = fmt.Errorf("failed removing account: %w", err)
		}
		return accountRemovalResult{state: state, err: err}
	}
}

func removalNavigation(state AccountState) navigation {
	return navigation{destination: accountRemoved, accounts: state.Accounts, account: state.Account, transactions: state.Transactions}
}

type removeAccountConfirmationScreen struct {
	previous      screen
	account       model.Account
	confirmation  confirmation
	removeAccount AccountRemover
	ctx           context.Context
}

func newRemoveAccountConfirmationScreen(previous screen, account model.Account, removeAccount AccountRemover, ctx context.Context, width, height int) *removeAccountConfirmationScreen {
	name := account.AccountNumber
	if account.BankCode != "" {
		name += "/" + account.BankCode
	}
	s := &removeAccountConfirmationScreen{
		previous: previous, account: account, removeAccount: removeAccount, ctx: ctx,
		confirmation: confirmation{title: "Remove account?", message: fmt.Sprintf("Remove %s and all of its locally stored data?", name)},
	}
	s.Resize(width, height)
	return s
}

func (s *removeAccountConfirmationScreen) Init() tea.Cmd { return nil }

func (s *removeAccountConfirmationScreen) Update(msg tea.Msg) (screen, tea.Cmd, navigation) {
	if result, ok := msg.(accountRemovalResult); ok {
		s.confirmation.loading = false
		if result.err != nil {
			s.confirmation.err = result.err
			return s, nil, navigation{}
		}
		return s, nil, removalNavigation(result.state)
	}
	switch s.confirmation.Update(msg) {
	case confirmationCancelled:
		return s.previous, nil, navigation{}
	case confirmationAccepted:
		if s.removeAccount == nil {
			s.confirmation.err = errors.New("account removal is unavailable")
			return s, nil, navigation{}
		}
		s.confirmation.loading = true
		return s, removeAccountCmd(s.ctx, s.removeAccount, s.account.AccountNumber), navigation{}
	}
	return s, nil, navigation{}
}

func (s *removeAccountConfirmationScreen) View() string { return s.confirmation.View() }
func (s *removeAccountConfirmationScreen) Resize(width, height int) {
	s.confirmation.width, s.confirmation.height = max(width, 1), max(height, 1)
}
