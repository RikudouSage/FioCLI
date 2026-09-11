package transactions

import (
	"fmt"
	"io"
	"strconv"

	"github.com/charmbracelet/x/ansi"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"go.chrastecky.dev/fio-client/fioclient/model"
	"go.chrastecky.dev/fio/fiocli/cmd/helper"
)

// RenderTable writes a compact, non-interactive transaction table.
func RenderTable(output io.Writer, txs []model.Transaction, limit int) {
	if limit > 0 && len(txs) > limit {
		txs = txs[:limit]
	}

	tbl := table.New(
		"ID",
		"Date",
		"Amount",
		"Counterparty",
		"Variable symbol",
		"Transaction type",
		"Comment",
		"Payer reference",
		"Local only",
	)
	tbl.WithHeaderFormatter(color.New(color.FgGreen, color.Underline).SprintfFunc())
	tbl.WithWidthFunc(ansi.StringWidth)
	tbl.WithWriter(output)
	for _, transaction := range txs {
		outgoing := transaction.TransactionType.IsOutgoing()
		transactionType := TranslateTransactionType(transaction.TransactionType.String())
		if outgoing {
			transactionType = color.New(color.FgRed).Sprint(transactionType)
		} else {
			transactionType = color.New(color.FgGreen).Sprint(transactionType)
		}

		tbl.AddRow(
			transaction.ID,
			transaction.Date.String(),
			fmt.Sprintf("%s %s", transaction.Amount, transaction.Currency),
			fmt.Sprintf("%s (%s/%s)", transaction.CounterpartyName, transaction.CounterpartyAccount, transaction.CounterpartyBankCode),
			pointerString(transaction.VariableSymbol),
			transactionType,
			pointerString(transaction.Comment),
			pointerString(transaction.PayerReference),
			helper.FormatBool(transaction.LocalOnly),
		)
	}
	tbl.Print()
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
