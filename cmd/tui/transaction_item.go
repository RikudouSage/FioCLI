package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"go.chrastecky.dev/fio-client/fioclient/model"
	transactionoutput "go.chrastecky.dev/fio/fiocli/cmd/transactions"
)

// transactionItem adapts a domain transaction to the list component.
type transactionItem struct{ transaction model.Transaction }

func (item transactionItem) displayName() string {
	name := item.transaction.CounterpartyName
	if name == "" {
		name = transactionoutput.TranslateTransactionType(item.transaction.TransactionType.String())
	}
	if name == "" {
		name = "Unknown transaction"
	}
	return name
}

func (item transactionItem) Title() string {
	return fmt.Sprintf("%s  %s", formatAmount(item.transaction), item.displayName())
}

func (item transactionItem) comment() string {
	return strings.TrimSpace(pointerString(item.transaction.Comment))
}

func (item transactionItem) Description() string {
	description := item.metadata()
	if paymentType := item.paymentType(); paymentType != "" {
		description += "  •  " + paymentType
	}
	if comment := item.comment(); comment != "" {
		description = comment + "  •  " + description
	}
	if item.transaction.LocalOnly {
		return "Pending confirmation  •  " + description
	}
	return description
}

func (item transactionItem) paymentType() string {
	rawType := strings.TrimSpace(item.transaction.TransactionType.String())
	translatedType := transactionoutput.TranslateTransactionType(rawType)
	title := strings.TrimSpace(item.displayName())
	if rawType == "" || strings.EqualFold(title, rawType) || strings.EqualFold(title, translatedType) {
		return ""
	}
	return translatedType
}

func (item transactionItem) metadata() string {
	account := counterpartyAccount(item.transaction)
	if account == "" {
		account = transactionoutput.TranslateTransactionType(item.transaction.TransactionType.String())
	}
	return item.transaction.Date.AsTime().Format("02 Jan 2006") + "  •  " + account
}

func (item transactionItem) FilterValue() string {
	return strings.Join([]string{
		formatAmount(item.transaction),
		item.transaction.Amount.Abs().StringFixed(2),
		item.transaction.CounterpartyName,
		item.transaction.CounterpartyAccount,
		item.transaction.CounterpartyBankName,
		transactionoutput.TranslateTransactionType(item.transaction.TransactionType.String()),
		item.transaction.TransactionType.String(),
		pointerString(item.transaction.Comment),
		pointerString(item.transaction.VariableSymbol),
		map[bool]string{true: "pending confirmation", false: ""}[item.transaction.LocalOnly],
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
