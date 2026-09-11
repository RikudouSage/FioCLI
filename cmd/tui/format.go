package tui

import (
	"strconv"
	"strings"

	"go.chrastecky.dev/fio-client/fioclient/model"
)

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
