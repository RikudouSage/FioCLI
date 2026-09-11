package transactions

import (
	"io"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/shopspring/decimal"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

// RenderDetail writes all available information about one transaction.
func RenderDetail(output io.Writer, account model.Account, transaction model.Transaction) {
	tbl := table.New("Field", "Value")
	tbl.WithWriter(output)
	tbl.WithWidthFunc(ansi.StringWidth)
	tbl.WithHeaderFormatter(color.New(color.FgGreen, color.Underline).SprintfFunc())
	tbl.WithFirstColumnFormatter(color.New(color.FgGreen, color.Bold).SprintfFunc())

	amount := formatMoney(transaction.Amount, transaction.Currency)
	if transaction.Amount.IsNegative() {
		amount = color.New(color.FgRed).Sprint(amount)
	} else if transaction.Amount.IsPositive() {
		amount = color.New(color.FgGreen).Sprint(amount)
	}
	transactionType := TranslateTransactionType(transaction.TransactionType.String())
	status := "Confirmed"
	if transaction.LocalOnly {
		status = color.New(color.FgYellow, color.Italic).Sprint("Pending confirmation")
	}

	rows := [][2]any{
		{"Transaction ID", transaction.ID},
		{"Status", status},
		{"Account", domesticAccount(account.AccountNumber, account.BankCode)},
		{"IBAN", formatIBAN(account.IBAN)},
		{"Date", transaction.Date.AsTime().Format("2 January 2006")},
		{"Amount", amount},
		{"Transaction type", valueOrDash(transactionType)},
		{"Counterparty", valueOrDash(transaction.CounterpartyName)},
		{"Counterparty account", valueOrDash(domesticAccount(transaction.CounterpartyAccount, transaction.CounterpartyBankCode))},
		{"Counterparty bank", valueOrDash(transaction.CounterpartyBankName)},
		{"BIC", valueOrDash(pointerString(transaction.BIC))},
		{"Variable symbol", valueOrDash(pointerString(transaction.VariableSymbol))},
		{"Constant symbol", valueOrDash(pointerString(transaction.ConstantSymbol))},
		{"Specific symbol", valueOrDash(pointerString(transaction.SpecificSymbol))},
		{"User identity", valueOrDash(pointerString(transaction.UserIdentity))},
		{"Performed by", valueOrDash(pointerString(transaction.PerformedBy))},
		{"Comment", valueOrDash(pointerString(transaction.Comment))},
		{"Additional info", valueOrDash(pointerString(transaction.AdditionalInfo))},
		{"Payer reference", valueOrDash(pointerString(transaction.PayerReference))},
		{"Instruction ID", valueOrDash(pointerInt64(transaction.InstructionID))},
	}
	for _, row := range rows {
		tbl.AddRow(row[0], row[1])
	}
	tbl.Print()
}

func formatMoney(amount decimal.Decimal, currency string) string {
	value := amount.StringFixed(2)
	sign := ""
	if strings.HasPrefix(value, "-") {
		sign, value = "-", strings.TrimPrefix(value, "-")
	}
	parts := strings.SplitN(value, ".", 2)
	for i := len(parts[0]) - 3; i > 0; i -= 3 {
		parts[0] = parts[0][:i] + "," + parts[0][i:]
	}
	return strings.TrimSpace(sign + strings.Join(parts, ".") + " " + currency)
}

func domesticAccount(accountNumber, bankCode string) string {
	accountNumber, bankCode = strings.TrimSpace(accountNumber), strings.TrimSpace(bankCode)
	if accountNumber == "" {
		return ""
	}
	if bankCode == "" {
		return accountNumber
	}
	return accountNumber + "/" + bankCode
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
	return valueOrDash(strings.Join(groups, " "))
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}
