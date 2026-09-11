package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.chrastecky.dev/fio-client/fioclient/model"
	transactionoutput "go.chrastecky.dev/fio/fiocli/cmd/transactions"
)

const detailHeaderHeight = 7

type transactionDetailScreen struct {
	transaction model.Transaction
	viewport    viewport.Model
	width       int
}

func newTransactionDetailScreen(transaction model.Transaction, width, height int) *transactionDetailScreen {
	s := &transactionDetailScreen{transaction: transaction, viewport: viewport.New(defaultWidth, defaultHeight-detailHeaderHeight)}
	s.Resize(width, height)
	s.viewport.GotoTop()
	return s
}

func (s *transactionDetailScreen) Init() tea.Cmd { return nil }

func (s *transactionDetailScreen) Update(msg tea.Msg) (screen, tea.Cmd, navigation) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "q", "ctrl+c":
			return s, nil, navigation{destination: quit}
		case "a":
			return s, nil, navigation{destination: showAccountPicker}
		case "esc", "backspace", "left", "h":
			return s, nil, navigation{destination: showTransactions}
		}
	}
	var cmd tea.Cmd
	s.viewport, cmd = s.viewport.Update(msg)
	return s, cmd, navigation{}
}

func (s *transactionDetailScreen) Resize(width, height int) {
	s.width = max(width, 1)
	s.viewport.Width = max(s.width-4, 1)
	s.viewport.Height = max(height-detailHeaderHeight, 1)
	s.viewport.SetContent(transactionDetails{transaction: s.transaction}.View())
}

func (s *transactionDetailScreen) View() string {
	header := titleStyle.Render("‹  Transaction details")
	help := helpStyle.Render("↑/k ↓/j scroll • a account • esc back • q quit")
	card := cardStyle.Width(max(s.width-cardStyle.GetHorizontalFrameSize(), 1)).Render(s.viewport.View())
	return header + "\n\n" + card + "\n" + help
}

// transactionDetails renders transaction data independently of navigation and scrolling.
type transactionDetails struct{ transaction model.Transaction }

func (d transactionDetails) View() string {
	t := d.transaction
	amountStyle := lipgloss.NewStyle().Bold(true).Foreground(outColor)
	if t.Amount.IsPositive() {
		amountStyle = amountStyle.Foreground(inColor)
	}
	rows := [][2]string{
		{"Date", t.Date.AsTime().Format("2 January 2006")},
		{"Type", transactionoutput.TranslateTransactionType(t.TransactionType.String())},
		{"Counterparty", t.CounterpartyName}, {"Counterparty account", counterpartyAccount(t)},
		{"Counterparty bank", t.CounterpartyBankName}, {"BIC", pointerString(t.BIC)},
		{"Variable symbol", pointerString(t.VariableSymbol)}, {"Constant symbol", pointerString(t.ConstantSymbol)},
		{"Specific symbol", pointerString(t.SpecificSymbol)}, {"User identity", pointerString(t.UserIdentity)},
		{"Performed by", pointerString(t.PerformedBy)}, {"Comment", pointerString(t.Comment)},
		{"Additional info", pointerString(t.AdditionalInfo)}, {"Payer reference", pointerString(t.PayerReference)},
		{"Status", map[bool]string{true: "Pending confirmation", false: "Confirmed"}[t.LocalOnly]},
		{"Instruction ID", pointerInt64(t.InstructionID)}, {"Transaction ID", strconv.FormatInt(t.ID, 10)}, {"Account", t.AccountNumber},
	}
	var content strings.Builder
	content.WriteString(amountStyle.Render(formatAmount(t)))
	content.WriteByte('\n')
	content.WriteString(titleStyle.Render(transactionItem{transaction: t}.displayName()))
	content.WriteByte('\n')
	content.WriteString(sectionStyle.Render("Payment information"))
	content.WriteByte('\n')
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
