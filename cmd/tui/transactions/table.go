package transactions

import (
	"io"

	"github.com/fatih/color"
	"github.com/rodaine/table"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

// RenderTable writes a compact, non-interactive transaction table.
func RenderTable(output io.Writer, txs []model.Transaction, limit int) {
	if limit > 0 && len(txs) > limit {
		txs = txs[:limit]
	}

	tbl := table.New(
		"ID",
		"Account number",
		"Date",
		"Amount",
		"Currency",
		"Counterparty account",
		"Counterparty name",
		"Counterparty bank code",
		"Counterparty bank name",
		"Constant symbol",
		"Variable symbol",
		"Specific symbol",
		"User identity",
		"Transaction type",
		"Performed by",
		"Additional info",
		"Comment",
		"BIC",
		"Instruction ID",
		"Payer reference",
	)
	tbl.WithHeaderFormatter(color.New(color.FgGreen, color.Underline).SprintfFunc())
	tbl.WithWriter(output)
	for _, transaction := range txs {
		tbl.AddRow(
			transaction.ID,
			transaction.AccountNumber,
			transaction.Date.String(),
			transaction.Amount.String(),
			transaction.Currency,
			transaction.CounterpartyAccount,
			transaction.CounterpartyName,
			transaction.CounterpartyBankCode,
			transaction.CounterpartyBankName,
			pointerString(transaction.ConstantSymbol),
			pointerString(transaction.VariableSymbol),
			pointerString(transaction.SpecificSymbol),
			pointerString(transaction.UserIdentity),
			translateTransactionType(transaction.TransactionType.String()),
			pointerString(transaction.PerformedBy),
			pointerString(transaction.AdditionalInfo),
			pointerString(transaction.Comment),
			pointerString(transaction.BIC),
			pointerInt64(transaction.InstructionID),
			pointerString(transaction.PayerReference),
		)
	}
	tbl.Print()
}
