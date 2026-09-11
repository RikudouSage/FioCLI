package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

const dashboardHeight = 5

type transactionsScreen struct {
	list      list.Model
	dashboard dashboard
}

func newTransactionsScreen(account model.Account, transactions []model.Transaction) *transactionsScreen {
	items := make([]list.Item, len(transactions))
	for index, transaction := range transactions {
		items[index] = transactionItem{transaction: transaction}
	}
	transactionList := list.New(items, newTransactionDelegate(), defaultWidth, defaultHeight)
	transactionList.SetShowTitle(false)
	transactionList.Filter = substringFilter
	transactionList.SetStatusBarItemName("transaction", "transactions")
	transactionList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details"))}
	}
	return &transactionsScreen{list: transactionList, dashboard: dashboard{account: account, transactions: transactions}}
}

func (s *transactionsScreen) Init() tea.Cmd { return nil }

func (s *transactionsScreen) Update(msg tea.Msg) (screen, tea.Cmd, navigation) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && !s.list.SettingFilter() {
		switch keyMsg.String() {
		case "ctrl+c", "q":
			return s, nil, navigation{destination: quit}
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

func (s *transactionsScreen) View() string { return s.dashboard.View(s.list.Width()) + s.list.View() }

func (s *transactionsScreen) Resize(width, height int) {
	s.list.SetSize(max(width, 1), max(height-dashboardHeight, 1))
}
