// Package tui provides the interactive terminal user interface.
package tui

import (
	"context"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

const (
	defaultWidth  = 80
	defaultHeight = 24
)

// Run starts the interactive account UI.
func Run(ctx context.Context, input io.Reader, output io.Writer, account model.Account, txs []model.Transaction, accounts []model.Account, switchAccount AccountSwitcher, reloadTransactions TransactionReloader, removeAccount AccountRemover) error {
	program := tea.NewProgram(newModelWithServices(ctx, account, txs, accounts, switchAccount, reloadTransactions, removeAccount), tea.WithContext(ctx), tea.WithInput(input), tea.WithOutput(output), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

// screen is the contract for a top-level view. The application shell owns
// navigation, while each screen owns only its local state and presentation.
type screen interface {
	Init() tea.Cmd
	Update(tea.Msg) (screen, tea.Cmd, navigation)
	View() string
	Resize(width, height int)
}

type navigation struct {
	destination  destination
	transaction  model.Transaction
	account      model.Account
	transactions []model.Transaction
	accounts     []model.Account
}

type destination uint8

const (
	stay destination = iota
	showTransactions
	showTransactionDetail
	showAccountPicker
	showRemoveAccountConfirmation
	selectAccount
	accountRemoved
	quit
)

// AccountSwitcher loads an account and its transactions. Persisting the
// selection remains the caller's responsibility.
type AccountSwitcher func(context.Context, string) (model.Account, []model.Transaction, error)

// TransactionReloader fetches the latest transactions for an account.
type TransactionReloader func(context.Context, string) ([]model.Transaction, error)

// AccountState is the usable application state after an account is removed.
type AccountState struct {
	Accounts     []model.Account
	Account      model.Account
	Transactions []model.Transaction
}

// AccountRemover removes an account and resolves the account that should be active afterwards.
type AccountRemover func(context.Context, string) (AccountState, error)

type tuiModel struct {
	current            screen
	transactions       *transactionsScreen
	width              int
	height             int
	ctx                context.Context
	accounts           []model.Account
	switchAccount      AccountSwitcher
	reloadTransactions TransactionReloader
	removeAccount      AccountRemover
}

func newModel(account model.Account, txs []model.Transaction) tuiModel {
	return newModelWithAccounts(context.Background(), account, txs, []model.Account{account}, nil)
}

func newModelWithAccounts(ctx context.Context, account model.Account, txs []model.Transaction, accounts []model.Account, switchAccount AccountSwitcher) tuiModel {
	return newModelWithServices(ctx, account, txs, accounts, switchAccount, nil, nil)
}

func newModelWithServices(ctx context.Context, account model.Account, txs []model.Transaction, accounts []model.Account, switchAccount AccountSwitcher, reloadTransactions TransactionReloader, removeAccount AccountRemover) tuiModel {
	transactions := newTransactionsScreenWithReload(account, txs, reloadTransactions, ctx)
	return tuiModel{current: transactions, transactions: transactions, width: defaultWidth, height: defaultHeight, ctx: ctx, accounts: accounts, switchAccount: switchAccount, reloadTransactions: reloadTransactions, removeAccount: removeAccount}
}

func (m tuiModel) Init() tea.Cmd { return m.current.Init() }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = max(size.Width, 1)
		m.height = max(size.Height, 1)
		m.current.Resize(m.width, m.height)
		return m, nil
	}

	updated, cmd, navigation := m.current.Update(msg)
	m.current = updated
	switch navigation.destination {
	case showTransactions:
		m.current = m.transactions
		m.current.Resize(m.width, m.height)
	case showTransactionDetail:
		m.current = newTransactionDetailScreen(navigation.transaction, m.width, m.height)
	case showAccountPicker:
		m.current = newAccountPickerScreen(m.current, m.accounts, m.transactions.dashboard.account.AccountNumber, m.switchAccount, m.removeAccount, m.ctx, m.width, m.height)
	case showRemoveAccountConfirmation:
		m.current = newRemoveAccountConfirmationScreen(m.current, navigation.account, m.removeAccount, m.ctx, m.width, m.height)
	case selectAccount:
		m.transactions = newTransactionsScreenWithReload(navigation.account, navigation.transactions, m.reloadTransactions, m.ctx)
		m.transactions.Resize(m.width, m.height)
		m.current = m.transactions
	case accountRemoved:
		m.accounts = navigation.accounts
		m.transactions = newTransactionsScreenWithReload(navigation.account, navigation.transactions, m.reloadTransactions, m.ctx)
		m.transactions.Resize(m.width, m.height)
		m.current = m.transactions
	case quit:
		return m, tea.Quit
	}
	return m, cmd
}

func (m tuiModel) View() string { return m.current.View() }
