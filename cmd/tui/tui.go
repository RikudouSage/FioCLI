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
func Run(ctx context.Context, input io.Reader, output io.Writer, account model.Account, txs []model.Transaction) error {
	program := tea.NewProgram(newModel(account, txs), tea.WithContext(ctx), tea.WithInput(input), tea.WithOutput(output), tea.WithAltScreen())
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
	destination destination
	transaction model.Transaction
}

type destination uint8

const (
	stay destination = iota
	showTransactions
	showTransactionDetail
	quit
)

type tuiModel struct {
	current      screen
	transactions *transactionsScreen
	width        int
	height       int
}

func newModel(account model.Account, txs []model.Transaction) tuiModel {
	transactions := newTransactionsScreen(account, txs)
	return tuiModel{current: transactions, transactions: transactions, width: defaultWidth, height: defaultHeight}
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
	case quit:
		return m, tea.Quit
	}
	return m, cmd
}

func (m tuiModel) View() string { return m.current.View() }
