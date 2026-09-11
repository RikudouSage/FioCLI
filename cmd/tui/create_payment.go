package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/shopspring/decimal"
	"go.chrastecky.dev/fio-api/fio/dto"
	fiotypes "go.chrastecky.dev/fio-api/fio/types"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

const (
	paymentAccountTo = iota
	paymentBankCode
	paymentAmount
	paymentCurrency
	paymentMessage
	paymentVariableSymbol
	paymentDate
	paymentType
	paymentConstantSymbol
	paymentSpecificSymbol
	paymentComment
)

type paymentField struct {
	label       string
	section     string
	placeholder string
	input       textinput.Model
	choices     []string
	selected    int
}

type paymentCreationResult struct{ err error }

var (
	paymentRequiredColor  = lipgloss.AdaptiveColor{Light: "#6941C6", Dark: "#B692F6"}
	paymentAdvancedColor  = lipgloss.AdaptiveColor{Light: "#175CD3", Dark: "#79B8FF"}
	paymentFormTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				Padding(0, 1).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(primaryColor)
)

// createPaymentScreen collects domestic-payment data in the order people
// usually need it: required details, common references, then advanced fields.
type createPaymentScreen struct {
	previous screen
	account  model.Account
	creator  DomesticPaymentCreator
	ctx      context.Context
	fields   []paymentField
	focused  int
	width    int
	height   int
	loading  bool
	err      error
}

func newCreatePaymentScreen(previous screen, account model.Account, creator DomesticPaymentCreator, ctx context.Context, width, height int) *createPaymentScreen {
	definitions := []struct {
		label, section, placeholder, value string
	}{
		{"Recipient account *", "Required", "e.g. 123456789", ""},
		{"Bank code *", "Required", "e.g. 2010", ""},
		{"Amount *", "Required", "e.g. 1250.00", ""},
		{"Currency", "Common", "", account.Currency},
		{"Message for recipient", "Common", "Optional message", ""},
		{"Variable symbol", "Common", "Optional reference", ""},
		{"Payment date", "Advanced", "YYYY-MM-DD (today if blank)", ""},
		{"Payment type", "Advanced", "standard, priority, or direct-debit", "standard"},
		{"Constant symbol", "Advanced", "Optional", ""},
		{"Specific symbol", "Advanced", "Optional", ""},
		{"Your comment", "Advanced", "Only visible to you", ""},
	}
	fields := make([]paymentField, len(definitions))
	for i, definition := range definitions {
		input := textinput.New()
		input.Prompt = ""
		input.Placeholder = ""
		// Some terminals (and terminal proxies) display ANSI styling emitted by
		// Bubble's cursor literally. The surrounding row already shows focus, so
		// keep the input itself deliberately unstyled and hide its block cursor.
		input.PromptStyle = lipgloss.NewStyle()
		input.TextStyle = lipgloss.NewStyle()
		input.PlaceholderStyle = lipgloss.NewStyle()
		input.Cursor.TextStyle = lipgloss.NewStyle()
		input.Cursor.SetMode(cursor.CursorHide)
		input.SetValue(definition.value)
		fields[i] = paymentField{label: definition.label, section: definition.section, placeholder: definition.placeholder, input: input}
	}
	fields[paymentCurrency].choices = []string{"CZK", "USD", "EUR", "PLN", "HUF", "GBP", "AUD", "CAD", "DKK", "CHF", "JPY", "NOK", "RUB", "SEK"}
	fields[paymentType].choices = []string{"standard", "priority", "direct-debit"}
	for index, currency := range fields[paymentCurrency].choices {
		if currency == strings.ToUpper(account.Currency) {
			fields[paymentCurrency].selected = index
			break
		}
	}
	if account.Currency != "" && fields[paymentCurrency].choices[fields[paymentCurrency].selected] != strings.ToUpper(account.Currency) {
		fields[paymentCurrency].choices = append([]string{strings.ToUpper(account.Currency)}, fields[paymentCurrency].choices...)
	}
	fields[0].input.Focus()
	return &createPaymentScreen{previous: previous, account: account, creator: creator, ctx: ctx, fields: fields, width: max(width, 1), height: max(height, 1)}
}

func (s *createPaymentScreen) Init() tea.Cmd          { return textinput.Blink }
func (s *createPaymentScreen) acceptsTextInput() bool { return !s.loading }

func (s *createPaymentScreen) Update(msg tea.Msg) (screen, tea.Cmd, navigation) {
	if result, ok := msg.(paymentCreationResult); ok {
		s.loading = false
		if result.err != nil {
			s.err = result.err
			return s, nil, navigation{}
		}
		if transactions, ok := s.previous.(*transactionsScreen); ok {
			status := transactions.list.NewStatusMessage("Payment created — refreshing local transactions…")
			refresh := transactions.refreshLocal()
			return transactions, tea.Batch(status, refresh), navigation{}
		}
		return s.previous, nil, navigation{}
	}
	if s.loading {
		return s, nil, navigation{}
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		s.err = nil
		if len(s.fields[s.focused].choices) > 0 {
			switch key.String() {
			case "left", "h":
				field := &s.fields[s.focused]
				field.selected = (field.selected + len(field.choices) - 1) % len(field.choices)
				return s, nil, navigation{}
			case "right", "l", "space":
				field := &s.fields[s.focused]
				field.selected = (field.selected + 1) % len(field.choices)
				return s, nil, navigation{}
			}
		}
		switch key.String() {
		case "esc":
			return s.previous, nil, navigation{}
		case "tab", "down":
			s.focus((s.focused + 1) % len(s.fields))
			return s, nil, navigation{}
		case "shift+tab", "up":
			s.focus((s.focused + len(s.fields) - 1) % len(s.fields))
			return s, nil, navigation{}
		case "enter":
			payment, err := s.payment()
			if err != nil {
				s.err = err
				return s, nil, navigation{}
			}
			if s.creator == nil {
				s.err = errors.New("payment creation is unavailable")
				return s, nil, navigation{}
			}
			s.loading = true
			return s, func() tea.Msg {
				err := s.creator(s.ctx, s.account.AccountNumber, payment)
				return paymentCreationResult{err: err}
			}, navigation{}
		}
	}
	var cmd tea.Cmd
	s.fields[s.focused].input, cmd = s.fields[s.focused].input.Update(msg)
	return s, cmd, navigation{}
}

func (s *createPaymentScreen) focus(index int) {
	s.fields[s.focused].input.Blur()
	s.focused = index
	s.fields[s.focused].input.Focus()
}

func (s *createPaymentScreen) payment() (dto.DomesticTransaction, error) {
	value := func(index int) string { return strings.TrimSpace(s.fields[index].input.Value()) }
	accountTo, bankCode := value(paymentAccountTo), value(paymentBankCode)
	if accountTo == "" {
		return dto.DomesticTransaction{}, errors.New("recipient account is required")
	}
	if bankCode == "" {
		return dto.DomesticTransaction{}, errors.New("bank code is required")
	}
	amount, err := decimal.NewFromString(value(paymentAmount))
	if err != nil || !amount.IsPositive() {
		return dto.DomesticTransaction{}, errors.New("amount must be greater than zero")
	}
	paymentType, err := paymentTypeFromInput(s.fields[paymentType].choices[s.fields[paymentType].selected])
	if err != nil {
		return dto.DomesticTransaction{}, err
	}
	payment := dto.DomesticTransaction{AccountTo: accountTo, BankCode: bankCode, Amount: amount, Currency: s.fields[paymentCurrency].choices[s.fields[paymentCurrency].selected], PaymentType: paymentType}
	if date := value(paymentDate); date == "" {
		payment.Date = fiotypes.Date(time.Now())
	} else if err := payment.Date.UnmarshalText([]byte(date)); err != nil {
		return dto.DomesticTransaction{}, fmt.Errorf("invalid payment date %q: %w", date, err)
	}
	payment.MessageForRecipient = optionalPaymentValue(value(paymentMessage))
	payment.VariableSymbol = optionalPaymentValue(value(paymentVariableSymbol))
	payment.ConstantSymbol = optionalPaymentValue(value(paymentConstantSymbol))
	payment.SpecificSymbol = optionalPaymentValue(value(paymentSpecificSymbol))
	payment.Comment = optionalPaymentValue(value(paymentComment))
	return payment, nil
}

func paymentTypeFromInput(value string) (dto.DomesticPaymentType, error) {
	switch strings.ToLower(value) {
	case "", "standard":
		return dto.DomesticPaymentStandard, nil
	case "priority":
		return dto.DomesticPaymentPriority, nil
	case "direct-debit":
		return dto.DomesticPaymentDirectDebit, nil
	default:
		return "", fmt.Errorf("invalid payment type %q", value)
	}
}

func optionalPaymentValue(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (s *createPaymentScreen) View() string {
	modalWidth := min(max(s.width-8, 42), 72)
	contentWidth := max(modalWidth-8, 1)
	lines := []string{
		paymentFormTitleStyle.Render("CREATE PAYMENT"),
		helpStyle.Render("From " + accountLabel(s.account)),
	}
	fieldLines := max(s.height-10-paymentErrorLines(s.err, contentWidth), 3)
	start, end := s.visibleFieldRange(fieldLines)
	if start > 0 {
		lines = append(lines, "", helpStyle.Render("↑ More fields above"))
	}
	lastSection := ""
	for index := start; index < end; index++ {
		field := &s.fields[index]
		if index == start || field.section != lastSection {
			lines = append(lines, "", paymentFormSectionStyle(field.section).Render(strings.ToUpper(field.section)+" FIELDS"))
			lastSection = field.section
		}
		field.input.Width = max(contentWidth-2, 1)
		field.input.Prompt = "  "
		label := paymentFieldLabel(field, false)
		if index == s.focused {
			field.input.Prompt = "> "
			label = paymentFieldLabel(field, true)
		}
		if len(field.choices) > 0 {
			lines = append(lines, "", label, paymentSelectView(field, index == s.focused))
			continue
		}
		lines = append(lines, "", label, paymentInputView(field))
	}
	if end < len(s.fields) {
		lines = append(lines, "", helpStyle.Render("↓ More fields below"))
	}
	if s.loading {
		lines = append(lines, "", helpStyle.Render("Creating payment…"))
	}
	if s.err != nil {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(outColor).Render(ansi.Wrap(s.err.Error(), contentWidth, "")))
	}
	help := "tab / ↑↓ next field  ·  enter create  ·  esc cancel"
	if len(s.fields[s.focused].choices) > 0 {
		help = "←/→ choose option  ·  tab / ↑↓ next field  ·  esc cancel"
	}
	lines = append(lines, "", helpStyle.Render(help))
	modal := cardStyle.Width(modalWidth - 4).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, modal)
}

// visibleFieldRange returns a contiguous part of the form that fits the
// terminal while keeping the focused field on screen. The layout is rendered
// directly rather than through a viewport so it remains readable in terminals
// too short for the complete payment form.
func (s *createPaymentScreen) visibleFieldRange(maxLines int) (int, int) {
	start := 0
	for {
		end := s.fieldRangeEnd(start, maxLines)
		if s.focused < end || start == s.focused {
			return start, end
		}
		start++
	}
}

func (s *createPaymentScreen) fieldRangeEnd(start, maxLines int) int {
	used, end, previousSection := 0, start, ""
	for end < len(s.fields) {
		field := s.fields[end]
		lines := 3 // blank line, label, and input/select value
		if end == start || field.section != previousSection {
			lines += 2 // blank line and section heading
		}
		if end > start && used+lines > maxLines {
			break
		}
		used += lines
		previousSection = field.section
		end++
	}
	return end
}

func paymentErrorLines(err error, width int) int {
	if err == nil {
		return 0
	}
	return lipgloss.Height(ansi.Wrap(err.Error(), max(width, 1), "")) + 1
}

func (s *createPaymentScreen) Resize(width, height int) {
	s.width, s.height = max(width, 1), max(height, 1)
}

func accountLabel(account model.Account) string {
	if account.BankCode == "" {
		return account.AccountNumber
	}
	return account.AccountNumber + "/" + account.BankCode
}

func paymentSelectView(field *paymentField, focused bool) string {
	value := field.choices[field.selected]
	if focused {
		return lipgloss.NewStyle().Bold(true).Foreground(paymentSectionColor(field.section)).Render("< " + value + " >")
	}
	return "  " + value + "  "
}

// paymentInputView renders placeholders outside bubbles/textinput. That keeps
// placeholder text visually distinct from a value and avoids its styled cursor
// being emitted literally by terminal proxies.
func paymentInputView(field *paymentField) string {
	if field.input.Value() == "" && field.placeholder != "" {
		return field.input.Prompt + helpStyle.Render(field.placeholder)
	}
	return field.input.View()
}

func paymentSectionColor(section string) lipgloss.AdaptiveColor {
	switch section {
	case "Required":
		return paymentRequiredColor
	case "Common":
		return pendingColor
	default:
		return paymentAdvancedColor
	}
}

func paymentFormSectionStyle(section string) lipgloss.Style {
	color := paymentSectionColor(section)
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(color)
}

func paymentFieldLabel(field *paymentField, focused bool) string {
	if focused {
		return lipgloss.NewStyle().Bold(true).Foreground(paymentSectionColor(field.section)).Render(field.label)
	}
	if before, ok := strings.CutSuffix(field.label, " *"); ok {
		return labelStyle.Render(before) + lipgloss.NewStyle().Foreground(paymentRequiredColor).Render(" *")
	}
	return labelStyle.Render(field.label)
}
