// Package tui provides the interactive terminal user interface.
package tui

import (
	"context"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"go.chrastecky.dev/fio-api/fio/dto"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

const (
	defaultWidth  = 80
	defaultHeight = 24
)

// Services contains the operations that back the interactive UI. Keeping these
// behind callbacks lets the UI start before the encrypted database is opened.
type Services struct {
	SwitchAccount      AccountSwitcher
	ReloadTransactions TransactionReloader
	LoadTransactions   TransactionReloader
	RemoveAccount      AccountRemover
	RegisterAccount    AccountRegistrar
	UnlockDatabase     DatabaseUnlocker
	CreatePayment      DomesticPaymentCreator
}

// Run starts the interactive account UI. When locked is true it starts on the
// database unlock screen instead of querying account data.
func Run(ctx context.Context, input io.Reader, output io.Writer, state AccountState, services Services, locked bool) error {
	program := tea.NewProgram(newModelWithState(ctx, state, services, locked), tea.WithContext(ctx), tea.WithInput(input), tea.WithOutput(output), tea.WithAltScreen())
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

// textInputScreen reports whether the current screen is actively receiving
// typed text. The application shell must not reserve single-key shortcuts in
// that state.
type textInputScreen interface {
	acceptsTextInput() bool
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
	showAddAccount
	showCreatePayment
	selectAccount
	accountRemoved
	accountsUpdated
	databaseUnlocked
	exitApplication
)

// AccountSwitcher loads an account and its transactions for the supplied
// account identifier. Persisting the selection remains the caller's
// responsibility.
type AccountSwitcher func(context.Context, string) (model.Account, []model.Transaction, error)

// TransactionReloader fetches the latest transactions for the supplied account
// identifier.
type TransactionReloader func(context.Context, string) ([]model.Transaction, error)

// AccountState contains the accounts and transactions to display after an
// account is removed.
type AccountState struct {
	// Accounts is the updated list of available accounts.
	Accounts []model.Account
	// Account is the account that should be active.
	Account model.Account
	// Transactions are the transactions belonging to Account.
	Transactions []model.Transaction
}

// AccountRemover removes the account identified by the supplied account
// identifier and resolves the account that should be active afterwards.
type AccountRemover func(context.Context, string) (AccountState, error)

// AccountRegistrar registers the supplied API key and returns the refreshed
// account list and the account that should be active.
type AccountRegistrar func(context.Context, string) ([]model.Account, model.Account, error)

// DatabaseUnlocker opens the database using password and returns the initial
// state to display after it has been opened.
type DatabaseUnlocker func(context.Context, string) (AccountState, error)

// DomesticPaymentCreator submits a domestic payment from the specified source
// account. The UI deliberately keeps this operation behind a callback so the
// screen remains independent of account storage and the Fio client.
type DomesticPaymentCreator func(context.Context, string, dto.DomesticTransaction) error

type tuiModel struct {
	current            screen
	transactions       *transactionsScreen
	width              int
	height             int
	ctx                context.Context
	accounts           []model.Account
	switchAccount      AccountSwitcher
	reloadTransactions TransactionReloader
	loadTransactions   TransactionReloader
	removeAccount      AccountRemover
	registerAccount    AccountRegistrar
	createPayment      DomesticPaymentCreator
}

func newModel(account model.Account, txs []model.Transaction) tuiModel {
	return newModelWithAccounts(context.Background(), account, txs, []model.Account{account}, nil)
}

func newModelWithAccounts(ctx context.Context, account model.Account, txs []model.Transaction, accounts []model.Account, switchAccount AccountSwitcher) tuiModel {
	return newModelWithServices(ctx, account, txs, accounts, switchAccount, nil, nil, nil)
}

func newModelWithServices(ctx context.Context, account model.Account, txs []model.Transaction, accounts []model.Account, switchAccount AccountSwitcher, reloadTransactions TransactionReloader, removeAccount AccountRemover, registerAccount AccountRegistrar) tuiModel {
	return newModelWithState(ctx, AccountState{Account: account, Transactions: txs, Accounts: accounts}, Services{SwitchAccount: switchAccount, ReloadTransactions: reloadTransactions, RemoveAccount: removeAccount, RegisterAccount: registerAccount}, false)
}

func newModelWithState(ctx context.Context, state AccountState, services Services, locked bool) tuiModel {
	account, txs, accounts := state.Account, state.Transactions, state.Accounts
	transactions := newTransactionsScreenWithReload(account, txs, services.ReloadTransactions, ctx)
	transactions.createPayment = services.CreatePayment
	transactions.loadTransactions = services.LoadTransactions
	model := tuiModel{current: transactions, transactions: transactions, width: defaultWidth, height: defaultHeight, ctx: ctx, accounts: accounts, switchAccount: services.SwitchAccount, reloadTransactions: services.ReloadTransactions, loadTransactions: services.LoadTransactions, removeAccount: services.RemoveAccount, registerAccount: services.RegisterAccount, createPayment: services.CreatePayment}
	if locked {
		model.current = newDatabaseUnlockScreen(services.UnlockDatabase, ctx, defaultWidth, defaultHeight)
		return model
	}
	if account.AccountNumber == "" {
		model.current = newAccountPickerScreen(transactions, accounts, "", services.SwitchAccount, services.RemoveAccount, services.RegisterAccount, ctx, defaultWidth, defaultHeight)
	}
	return model
}

func (m tuiModel) Init() tea.Cmd { return m.current.Init() }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if input, ok := m.current.(textInputScreen); ok && input.acceptsTextInput() {
				break
			}
			return m, tea.Quit
		}
	}
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
		m.current = newAccountPickerScreen(m.current, m.accounts, m.transactions.dashboard.account.AccountNumber, m.switchAccount, m.removeAccount, m.registerAccount, m.ctx, m.width, m.height)
	case showRemoveAccountConfirmation:
		m.current = newRemoveAccountConfirmationScreen(m.current, navigation.account, m.removeAccount, m.ctx, m.width, m.height)
	case showAddAccount:
		m.current = newAddAccountScreen(m.current, m.registerAccount, m.ctx, m.width, m.height)
		cmd = m.current.Init()
	case showCreatePayment:
		m.current = newCreatePaymentScreen(m.transactions, m.transactions.dashboard.account, m.createPayment, m.ctx, m.width, m.height)
		cmd = m.current.Init()
	case selectAccount:
		m.transactions = newTransactionsScreenWithReload(navigation.account, navigation.transactions, m.reloadTransactions, m.ctx)
		m.transactions.createPayment = m.createPayment
		m.transactions.loadTransactions = m.loadTransactions
		m.transactions.Resize(m.width, m.height)
		m.current = m.transactions
	case accountRemoved:
		m.accounts = navigation.accounts
		m.transactions = newTransactionsScreenWithReload(navigation.account, navigation.transactions, m.reloadTransactions, m.ctx)
		m.transactions.createPayment = m.createPayment
		m.transactions.loadTransactions = m.loadTransactions
		m.transactions.Resize(m.width, m.height)
		m.current = m.transactions
	case accountsUpdated:
		m.accounts = navigation.accounts
	case databaseUnlocked:
		m.accounts = navigation.accounts
		m.transactions = newTransactionsScreenWithReload(navigation.account, navigation.transactions, m.reloadTransactions, m.ctx)
		m.transactions.createPayment = m.createPayment
		m.transactions.loadTransactions = m.loadTransactions
		m.transactions.Resize(m.width, m.height)
		if navigation.account.AccountNumber == "" {
			m.current = newAccountPickerScreen(m.transactions, m.accounts, "", m.switchAccount, m.removeAccount, m.registerAccount, m.ctx, m.width, m.height)
		} else {
			m.current = m.transactions
		}
	case exitApplication:
		return m, tea.Quit
	}
	return m, cmd
}

func (m tuiModel) View() string { return m.current.View() }
