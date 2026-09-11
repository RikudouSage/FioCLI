package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

const dashboardHeight = 5

type transactionsScreen struct {
	list               list.Model
	dashboard          dashboard
	reloadTransactions TransactionReloader
	loadTransactions   TransactionReloader
	ctx                context.Context
	reloading          bool
	createPayment      DomesticPaymentCreator
}

func newTransactionsScreen(account model.Account, transactions []model.Transaction) *transactionsScreen {
	return newTransactionsScreenWithReload(account, transactions, nil, context.Background())
}

func newTransactionsScreenWithReload(account model.Account, transactions []model.Transaction, reloadTransactions TransactionReloader, ctx context.Context) *transactionsScreen {
	transactionList := list.New(transactionItems(transactions), newTransactionDelegate(), defaultWidth, defaultHeight)
	transactionList.SetShowTitle(false)
	transactionList.Filter = substringFilter
	transactionList.SetStatusBarItemName("transaction", "transactions")
	transactionList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
			key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "payment")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "account")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		}
	}
	return &transactionsScreen{list: transactionList, dashboard: dashboard{account: account, transactions: transactions}, reloadTransactions: reloadTransactions, ctx: ctx}
}

func (s *transactionsScreen) Init() tea.Cmd { return nil }

func (s *transactionsScreen) acceptsTextInput() bool { return s.list.SettingFilter() }

func (s *transactionsScreen) Update(msg tea.Msg) (screen, tea.Cmd, navigation) {
	if result, ok := msg.(transactionReloadResult); ok {
		s.reloading = false
		s.dashboard.activity = ""
		s.list.StopSpinner()
		if result.err != nil {
			return s, s.list.NewStatusMessage(fmt.Sprintf("Reload failed: %v", result.err)), navigation{}
		}
		items := transactionItems(result.transactions)
		s.dashboard.transactions = result.transactions
		return s, s.list.SetItems(items), navigation{}
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok && !s.list.SettingFilter() {
		switch keyMsg.String() {
		case "a":
			return s, nil, navigation{destination: showAccountPicker}
		case "p":
			return s, nil, navigation{destination: showCreatePayment}
		case "r":
			return s, s.refresh(), navigation{}
		case "enter":
			if item, ok := s.list.SelectedItem().(transactionItem); ok {
				return s, nil, navigation{destination: showTransactionDetail, transaction: item.transaction}
			}
			return s, nil, navigation{}
		}
	}
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd, navigation{}
}

// refresh synchronizes the account and returns the updated locally stored
// transactions. It is shared by the manual shortcut and payment submission.
func (s *transactionsScreen) refresh() tea.Cmd {
	if s.reloading {
		return nil
	}
	if s.reloadTransactions == nil {
		return s.list.NewStatusMessage("Reload is unavailable")
	}
	s.reloading = true
	s.dashboard.activity = "⟳ Syncing transactions…"
	spinner := s.list.StartSpinner()
	accountNumber := s.dashboard.account.AccountNumber
	reload := func() tea.Msg {
		transactions, err := s.reloadTransactions(s.ctx, accountNumber)
		return transactionReloadResult{transactions: transactions, err: err}
	}
	return tea.Batch(spinner, reload)
}

// refreshLocal reloads only the transactions already stored in the local
// database. Unlike refresh, it never contacts Fio or synchronizes new data.
func (s *transactionsScreen) refreshLocal() tea.Cmd {
	if s.reloading {
		return nil
	}
	if s.loadTransactions == nil {
		return s.list.NewStatusMessage("Local refresh is unavailable")
	}
	s.reloading = true
	s.dashboard.activity = "⟳ Refreshing local transactions…"
	spinner := s.list.StartSpinner()
	accountNumber := s.dashboard.account.AccountNumber
	load := func() tea.Msg {
		transactions, err := s.loadTransactions(s.ctx, accountNumber)
		return transactionReloadResult{transactions: transactions, err: err}
	}
	return tea.Batch(spinner, load)
}

type transactionReloadResult struct {
	transactions []model.Transaction
	err          error
}

func transactionItems(transactions []model.Transaction) []list.Item {
	items := make([]list.Item, len(transactions))
	for index, transaction := range transactions {
		items[index] = transactionItem{transaction: transaction}
	}
	return items
}

func (s *transactionsScreen) View() string { return s.dashboard.View(s.list.Width()) + s.list.View() }

func (s *transactionsScreen) Resize(width, height int) {
	s.list.SetSize(max(width, 1), max(height-dashboardHeight, 1))
}
