package transactions

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

const (
	defaultWidth       = 80
	defaultHeight      = 24
	detailHeaderHeight = 4
)

type screen uint8

const (
	listScreen screen = iota
	detailScreen
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	labelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7D7AFF"})
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B6B6B", Dark: "#9B9B9B"})
)

// Run renders transactions as an interactive list with a detail view.
func Run(ctx context.Context, input io.Reader, output io.Writer, txs []model.Transaction) error {
	program := tea.NewProgram(
		newModel(txs),
		tea.WithContext(ctx),
		tea.WithInput(input),
		tea.WithOutput(output),
		tea.WithAltScreen(),
	)
	_, err := program.Run()
	return err
}

type transactionItem struct {
	transaction model.Transaction
}

type transactionDelegate struct {
	normalTitle   lipgloss.Style
	normalDesc    lipgloss.Style
	selectedTitle lipgloss.Style
	selectedDesc  lipgloss.Style
}

func newTransactionDelegate() transactionDelegate {
	styles := list.NewDefaultItemStyles()
	return transactionDelegate{
		normalTitle:   styles.NormalTitle,
		normalDesc:    styles.NormalDesc,
		selectedTitle: styles.SelectedTitle,
		selectedDesc:  styles.SelectedDesc,
	}
}

func (receiver transactionDelegate) Height() int  { return 2 }
func (receiver transactionDelegate) Spacing() int { return 1 }
func (receiver transactionDelegate) Update(tea.Msg, *list.Model) tea.Cmd {
	return nil
}

func (receiver transactionDelegate) Render(writer io.Writer, transactionList list.Model, index int, item list.Item) {
	transaction, ok := item.(transactionItem)
	if !ok || transactionList.Width() <= 0 {
		return
	}

	titleStyle, descriptionStyle := receiver.normalTitle, receiver.normalDesc
	if index == transactionList.Index() && transactionList.FilterState() != list.Filtering {
		titleStyle, descriptionStyle = receiver.selectedTitle, receiver.selectedDesc
	}

	contentWidth := max(transactionList.Width()-titleStyle.GetPaddingLeft()-titleStyle.GetPaddingRight(), 1)
	title := ansi.Truncate(transaction.Title(), contentWidth, "…")
	description := ansi.Truncate(transaction.Description(), contentWidth, "…")
	fmt.Fprintf(writer, "%s\n%s", titleStyle.Render(title), descriptionStyle.Render(description))
}

func (receiver transactionItem) Title() string {
	name := receiver.transaction.CounterpartyName
	if name == "" {
		name = receiver.transaction.TransactionType.String()
	}
	if name == "" {
		name = "Unknown transaction"
	}
	return fmt.Sprintf("%s  %s", formatAmount(receiver.transaction), name)
}

func (receiver transactionItem) Description() string {
	account := counterpartyAccount(receiver.transaction)
	if account == "" {
		account = receiver.transaction.TransactionType.String()
	}
	return fmt.Sprintf("%s  %s", receiver.transaction.Date.AsTime().Format("02 Jan 2006"), account)
}

func (receiver transactionItem) FilterValue() string {
	return strings.Join([]string{
		formatAmount(receiver.transaction),
		receiver.transaction.Amount.Abs().StringFixed(2),
		receiver.transaction.CounterpartyName,
		receiver.transaction.CounterpartyAccount,
		receiver.transaction.CounterpartyBankName,
		receiver.transaction.TransactionType.String(),
		pointerString(receiver.transaction.Comment),
		pointerString(receiver.transaction.VariableSymbol),
	}, " ")
}

func substringFilter(term string, targets []string) []list.Rank {
	term = strings.ToLower(term)
	ranks := make([]list.Rank, 0, len(targets))
	for index, target := range targets {
		if strings.Contains(strings.ToLower(target), term) {
			ranks = append(ranks, list.Rank{Index: index})
		}
	}
	return ranks
}

type tuiModel struct {
	list     list.Model
	viewport viewport.Model
	screen   screen
	selected model.Transaction
	width    int
	height   int
}

func newModel(txs []model.Transaction) tuiModel {
	items := make([]list.Item, len(txs))
	for i, transaction := range txs {
		items[i] = transactionItem{transaction: transaction}
	}

	transactionList := list.New(items, newTransactionDelegate(), defaultWidth, defaultHeight)
	transactionList.Title = "Transactions"
	transactionList.Filter = substringFilter
	transactionList.SetStatusBarItemName("transaction", "transactions")
	transactionList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details"))}
	}

	return tuiModel{
		list:     transactionList,
		viewport: viewport.New(defaultWidth, defaultHeight-detailHeaderHeight),
		width:    defaultWidth,
		height:   defaultHeight,
	}
}

func (receiver tuiModel) Init() tea.Cmd { return nil }

func (receiver tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		receiver.resize(msg.Width, msg.Height)
	case tea.KeyMsg:
		if receiver.screen == detailScreen {
			switch msg.String() {
			case "q", "ctrl+c":
				return receiver, tea.Quit
			case "esc", "backspace", "left", "h":
				receiver.screen = listScreen
				return receiver, nil
			}
		} else if !receiver.list.SettingFilter() {
			switch msg.String() {
			case "ctrl+c", "q":
				return receiver, tea.Quit
			case "enter":
				if item, ok := receiver.list.SelectedItem().(transactionItem); ok {
					receiver.selected = item.transaction
					receiver.screen = detailScreen
					receiver.viewport.SetContent(receiver.detailContent())
					receiver.viewport.GotoTop()
				}
				return receiver, nil
			}
		}
	}

	var cmd tea.Cmd
	if receiver.screen == detailScreen {
		receiver.viewport, cmd = receiver.viewport.Update(msg)
	} else {
		receiver.list, cmd = receiver.list.Update(msg)
	}
	return receiver, cmd
}

func (receiver *tuiModel) resize(width, height int) {
	receiver.width = max(width, 1)
	receiver.height = max(height, 1)
	receiver.list.SetSize(receiver.width, receiver.height)
	receiver.viewport.Width = receiver.width
	receiver.viewport.Height = max(receiver.height-detailHeaderHeight, 1)
	if receiver.screen == detailScreen {
		receiver.viewport.SetContent(receiver.detailContent())
	}
}

func (receiver tuiModel) View() string {
	if receiver.screen == listScreen {
		return receiver.list.View()
	}
	header := titleStyle.Render("Transaction details")
	help := helpStyle.Render("↑/k ↓/j scroll • esc back • q quit")
	return header + "\n" + receiver.viewport.View() + "\n" + help
}

func (receiver tuiModel) detailContent() string {
	t := receiver.selected
	rows := [][2]string{
		{"Amount", formatAmount(t)},
		{"Date", t.Date.AsTime().Format("2 January 2006")},
		{"Type", t.TransactionType.String()},
		{"Counterparty", t.CounterpartyName},
		{"Counterparty account", counterpartyAccount(t)},
		{"Counterparty bank", t.CounterpartyBankName},
		{"BIC", pointerString(t.BIC)},
		{"Variable symbol", pointerString(t.VariableSymbol)},
		{"Constant symbol", pointerString(t.ConstantSymbol)},
		{"Specific symbol", pointerString(t.SpecificSymbol)},
		{"User identity", pointerString(t.UserIdentity)},
		{"Performed by", pointerString(t.PerformedBy)},
		{"Comment", pointerString(t.Comment)},
		{"Additional info", pointerString(t.AdditionalInfo)},
		{"Payer reference", pointerString(t.PayerReference)},
		{"Instruction ID", pointerInt64(t.InstructionID)},
		{"Transaction ID", strconv.FormatInt(t.ID, 10)},
		{"Account", t.AccountNumber},
	}

	var content strings.Builder
	for _, row := range rows {
		value := row[1]
		if value == "" {
			value = "—"
		}
		content.WriteString(labelStyle.Width(21).Render(row[0]))
		content.WriteString(" ")
		content.WriteString(value)
		content.WriteByte('\n')
	}
	return strings.TrimSuffix(content.String(), "\n")
}

func formatAmount(transaction model.Transaction) string {
	return transaction.Amount.StringFixed(2) + " " + transaction.Currency
}

func counterpartyAccount(transaction model.Transaction) string {
	if transaction.CounterpartyAccount == "" {
		return ""
	}
	if transaction.CounterpartyBankCode == "" {
		return transaction.CounterpartyAccount
	}
	return transaction.CounterpartyAccount + "/" + transaction.CounterpartyBankCode
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func pointerInt64(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}
