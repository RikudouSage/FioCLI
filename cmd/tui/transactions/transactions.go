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
	"github.com/shopspring/decimal"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

const (
	defaultWidth       = 80
	defaultHeight      = 24
	dashboardHeight    = 5
	detailHeaderHeight = 7
)

type screen uint8

const (
	listScreen screen = iota
	detailScreen
)

var (
	primaryColor = lipgloss.AdaptiveColor{Light: "#075E54", Dark: "#5EE1B7"}
	mutedColor   = lipgloss.AdaptiveColor{Light: "#667085", Dark: "#8B95A5"}
	faintColor   = lipgloss.AdaptiveColor{Light: "#98A2B3", Dark: "#667085"}
	pendingColor = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#E3B341"}
	inColor      = lipgloss.AdaptiveColor{Light: "#08783E", Dark: "#57D68D"}
	outColor     = lipgloss.AdaptiveColor{Light: "#B42318", Dark: "#FF7B72"}

	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	labelStyle   = lipgloss.NewStyle().Foreground(mutedColor)
	helpStyle    = lipgloss.NewStyle().Foreground(mutedColor)
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(primaryColor).MarginTop(1)
	cardStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(mutedColor).Padding(0, 2)
)

// Run renders transactions as an interactive list with a detail view.
func Run(ctx context.Context, input io.Reader, output io.Writer, account model.Account, txs []model.Transaction) error {
	program := tea.NewProgram(
		newModel(account, txs),
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
	return transactionDelegate{
		normalTitle:   lipgloss.NewStyle().PaddingLeft(2),
		normalDesc:    lipgloss.NewStyle().PaddingLeft(2),
		selectedTitle: lipgloss.NewStyle().Border(lipgloss.ThickBorder(), false, false, false, true).BorderForeground(primaryColor).PaddingLeft(1).Bold(true),
		selectedDesc:  lipgloss.NewStyle().Border(lipgloss.ThickBorder(), false, false, false, true).BorderForeground(primaryColor).PaddingLeft(1),
	}
}

func (receiver transactionDelegate) Height() int  { return 7 }
func (receiver transactionDelegate) Spacing() int { return 0 }
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
	pending := transaction.transaction.LocalOnly
	titleStyle = withPendingStyle(titleStyle, pending)
	descriptionStyle = withPendingStyle(descriptionStyle, pending)

	contentWidth := max(transactionList.Width()-titleStyle.GetHorizontalFrameSize(), 1)
	monthHeader := ""
	if isMonthStart(transactionList, index, transaction.transaction) {
		month := transaction.transaction.Date.AsTime().Format("January 2006")
		incoming, outgoing, currency := monthTotals(transactionList, transaction.transaction)
		monthHeader = renderMonthHeader(month, incoming, outgoing, currency, transactionList.Width())
	}
	amount := formatAmount(transaction.transaction)
	nameWidth := max(contentWidth-lipgloss.Width(amount)-2, 1)
	name := ansi.Truncate(transaction.displayName(), nameWidth, "…")
	space := strings.Repeat(" ", max(contentWidth-lipgloss.Width(name)-lipgloss.Width(amount), 1))
	amountStyle := lipgloss.NewStyle().Foreground(outColor).Italic(pending)
	if transaction.transaction.Amount.IsPositive() {
		amountStyle = amountStyle.Foreground(inColor)
	}
	title := name + space + amountStyle.Render(amount)
	metadata := lipgloss.NewStyle().Foreground(mutedColor).Italic(pending).Render(transaction.metadata())
	if paymentType := transaction.paymentType(); paymentType != "" {
		metadata += lipgloss.NewStyle().Foreground(faintColor).Italic(pending).Render("  •  " + paymentType)
	}
	description := metadata
	if comment := transaction.comment(); comment != "" {
		description = lipgloss.NewStyle().Bold(true).Italic(pending).Foreground(primaryColor).Render(comment) +
			lipgloss.NewStyle().Foreground(mutedColor).Italic(pending).Render("  •  ") + metadata
	}
	if pending {
		description = lipgloss.NewStyle().Bold(true).Italic(true).Foreground(pendingColor).Render("Pending confirmation") +
			lipgloss.NewStyle().Foreground(mutedColor).Italic(true).Render("  •  ") + description
	}
	description = ansi.Truncate(description, contentWidth, "…")
	if monthHeader != "" {
		fmt.Fprintf(writer, "%s\n%s\n%s", monthHeader, titleStyle.Render(title), descriptionStyle.Render(description))
		if index == len(transactionList.VisibleItems())-1 {
			fmt.Fprintf(writer, "\n%s", endOfTransactions(transactionList.Width()))
		}
		return
	}
	fmt.Fprintf(writer, "\n%s\n%s", titleStyle.Render(title), descriptionStyle.Render(description))
	if index == len(transactionList.VisibleItems())-1 {
		fmt.Fprintf(writer, "\n%s", endOfTransactions(transactionList.Width()))
	}
}

func endOfTransactions(width int) string {
	message := "No more transactions"
	return lipgloss.NewStyle().Foreground(faintColor).Render(
		lipgloss.PlaceHorizontal(max(width, 1), lipgloss.Center, message),
	)
}

func renderMonthHeader(month, incoming, outgoing, currency string, width int) string {
	indent := "  "
	title := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(strings.ToUpper(month))
	label := lipgloss.NewStyle().Foreground(mutedColor)
	incomingValue := lipgloss.NewStyle().Bold(true).Foreground(inColor).Render("+" + incoming + " " + currency)
	outgoingValue := lipgloss.NewStyle().Bold(true).Foreground(outColor).Render("−" + outgoing + " " + currency)
	stats := label.Render("Incoming ") + incomingValue + label.Render("    Outgoing ") + outgoingValue
	divider := lipgloss.NewStyle().Foreground(faintColor).Render(strings.Repeat("─", max(width-len(indent), 1)))
	return "\n" + indent + title + "\n" + indent + stats + "\n" + indent + divider
}

func monthTotals(transactionList list.Model, transaction model.Transaction) (string, string, string) {
	month := transaction.Date.AsTime()
	currency := transaction.Currency
	incoming, outgoing := decimal.Zero, decimal.Zero
	for _, item := range transactionList.Items() {
		candidate, ok := item.(transactionItem)
		if !ok || candidate.transaction.Currency != currency {
			continue
		}
		date := candidate.transaction.Date.AsTime()
		if date.Year() != month.Year() || date.Month() != month.Month() {
			continue
		}
		if candidate.transaction.Amount.IsPositive() {
			incoming = incoming.Add(candidate.transaction.Amount)
		} else {
			outgoing = outgoing.Add(candidate.transaction.Amount.Abs())
		}
	}
	return incoming.StringFixed(2), outgoing.StringFixed(2), currency
}

func isMonthStart(transactionList list.Model, index int, transaction model.Transaction) bool {
	start, _ := transactionList.Paginator.GetSliceBounds(len(transactionList.VisibleItems()))
	if index == start || index == 0 {
		return true
	}
	items := transactionList.VisibleItems()
	if index > len(items)-1 {
		return false
	}
	previous, ok := items[index-1].(transactionItem)
	if !ok {
		return true
	}
	date, previousDate := transaction.Date.AsTime(), previous.transaction.Date.AsTime()
	return date.Year() != previousDate.Year() || date.Month() != previousDate.Month()
}

func withPendingStyle(style lipgloss.Style, pending bool) lipgloss.Style {
	return style.Italic(pending)
}

func (receiver transactionItem) displayName() string {
	name := receiver.transaction.CounterpartyName
	if name == "" {
		name = translateTransactionType(receiver.transaction.TransactionType.String())
	}
	if name == "" {
		name = "Unknown transaction"
	}
	return name
}

func (receiver transactionItem) Title() string {
	return fmt.Sprintf("%s  %s", formatAmount(receiver.transaction), receiver.displayName())
}

func (receiver transactionItem) comment() string {
	return strings.TrimSpace(pointerString(receiver.transaction.Comment))
}

func (receiver transactionItem) Description() string {
	description := receiver.metadata()
	if paymentType := receiver.paymentType(); paymentType != "" {
		description += "  •  " + paymentType
	}
	if comment := receiver.comment(); comment != "" {
		description = comment + "  •  " + description
	}
	if receiver.transaction.LocalOnly {
		return "Pending confirmation  •  " + description
	}
	return description
}

func (receiver transactionItem) paymentType() string {
	rawType := strings.TrimSpace(receiver.transaction.TransactionType.String())
	translatedType := translateTransactionType(rawType)
	title := strings.TrimSpace(receiver.displayName())
	if rawType == "" || strings.EqualFold(title, rawType) || strings.EqualFold(title, translatedType) {
		return ""
	}
	return translatedType
}

func (receiver transactionItem) metadata() string {
	account := counterpartyAccount(receiver.transaction)
	if account == "" {
		account = translateTransactionType(receiver.transaction.TransactionType.String())
	}
	return receiver.transaction.Date.AsTime().Format("02 Jan 2006") + "  •  " + account
}

func (receiver transactionItem) FilterValue() string {
	return strings.Join([]string{
		formatAmount(receiver.transaction),
		receiver.transaction.Amount.Abs().StringFixed(2),
		receiver.transaction.CounterpartyName,
		receiver.transaction.CounterpartyAccount,
		receiver.transaction.CounterpartyBankName,
		translateTransactionType(receiver.transaction.TransactionType.String()),
		receiver.transaction.TransactionType.String(),
		pointerString(receiver.transaction.Comment),
		pointerString(receiver.transaction.VariableSymbol),
		map[bool]string{true: "pending confirmation", false: ""}[receiver.transaction.LocalOnly],
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
	txs      []model.Transaction
	account  model.Account
}

func newModel(account model.Account, txs []model.Transaction) tuiModel {
	items := make([]list.Item, len(txs))
	for i, transaction := range txs {
		items[i] = transactionItem{transaction: transaction}
	}

	transactionList := list.New(items, newTransactionDelegate(), defaultWidth, defaultHeight)
	transactionList.SetShowTitle(false)
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
		txs:      txs,
		account:  account,
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
	receiver.list.SetSize(receiver.width, max(receiver.height-dashboardHeight, 1))
	receiver.viewport.Width = max(receiver.width-4, 1)
	receiver.viewport.Height = max(receiver.height-detailHeaderHeight, 1)
	if receiver.screen == detailScreen {
		receiver.viewport.SetContent(receiver.detailContent())
	}
}

func (receiver tuiModel) View() string {
	if receiver.screen == listScreen {
		return receiver.dashboard() + receiver.list.View()
	}
	header := titleStyle.Render("‹  Transaction details")
	help := helpStyle.Render("↑/k ↓/j scroll • esc back • q quit")
	card := cardStyle.Width(max(receiver.width-cardStyle.GetHorizontalFrameSize(), 1)).Render(receiver.viewport.View())
	return header + "\n\n" + card + "\n" + help
}

func (receiver tuiModel) dashboard() string {
	accountNumber := receiver.account.AccountNumber
	if receiver.account.BankCode != "" {
		accountNumber += "/" + receiver.account.BankCode
	}
	if accountNumber == "" {
		accountNumber = "No account"
	}

	incoming, outgoing := "0.00", "0.00"
	currency := receiver.account.Currency
	if currency == "" && len(receiver.txs) > 0 {
		currency = receiver.txs[0].Currency
	}
	if len(receiver.txs) > 0 {
		in, out := decimal.Zero, decimal.Zero
		for _, transaction := range receiver.txs {
			if transaction.Currency != currency {
				continue
			}
			if transaction.Amount.IsPositive() {
				in = in.Add(transaction.Amount)
			} else {
				out = out.Add(transaction.Amount.Abs())
			}
		}
		incoming, outgoing = in.StringFixed(2), out.StringFixed(2)
	}

	heading := titleStyle.Render("Fio Account") + "  " + helpStyle.Render(accountNumber)
	subtitle := "Recent activity"
	if receiver.account.IBAN != "" {
		subtitle = formatIBAN(receiver.account.IBAN)
	}
	summary := lipgloss.NewStyle().Foreground(inColor).Render("↓ "+incoming+" "+currency) +
		"    " + lipgloss.NewStyle().Foreground(outColor).Render("↑ "+outgoing+" "+currency)
	divider := lipgloss.NewStyle().Foreground(mutedColor).Render(strings.Repeat("─", max(receiver.width, 1)))
	return heading + "\n" + helpStyle.Render(subtitle) + "\n" + summary + "\n" + divider + "\n"
}

func formatIBAN(iban string) string {
	compact := strings.Join(strings.Fields(iban), "")
	groups := make([]string, 0, (len(compact)+3)/4)
	for len(compact) > 4 {
		groups = append(groups, compact[:4])
		compact = compact[4:]
	}
	if compact != "" {
		groups = append(groups, compact)
	}
	return strings.Join(groups, " ")
}

func (receiver tuiModel) detailContent() string {
	t := receiver.selected
	amountStyle := lipgloss.NewStyle().Bold(true).Foreground(outColor)
	if t.Amount.IsPositive() {
		amountStyle = amountStyle.Foreground(inColor)
	}
	rows := [][2]string{
		{"Date", t.Date.AsTime().Format("2 January 2006")},
		{"Type", translateTransactionType(t.TransactionType.String())},
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
		{"Status", map[bool]string{true: "Pending confirmation", false: "Confirmed"}[t.LocalOnly]},
		{"Instruction ID", pointerInt64(t.InstructionID)},
		{"Transaction ID", strconv.FormatInt(t.ID, 10)},
		{"Account", t.AccountNumber},
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
