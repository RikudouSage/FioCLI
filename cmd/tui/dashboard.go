package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/shopspring/decimal"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

// dashboard is a reusable, stateless summary component.
type dashboard struct {
	account      model.Account
	transactions []model.Transaction
}

func (d dashboard) View(width int) string {
	accountNumber := d.account.AccountNumber
	if d.account.BankCode != "" {
		accountNumber += "/" + d.account.BankCode
	}
	if accountNumber == "" {
		accountNumber = "No account"
	}

	currency := d.account.Currency
	if currency == "" && len(d.transactions) > 0 {
		currency = d.transactions[0].Currency
	}
	incoming, outgoing := decimal.Zero, decimal.Zero
	for _, transaction := range d.transactions {
		if transaction.Currency != currency {
			continue
		}
		if transaction.Amount.IsPositive() {
			incoming = incoming.Add(transaction.Amount)
		} else {
			outgoing = outgoing.Add(transaction.Amount.Abs())
		}
	}

	heading := titleStyle.Render("Fio Account") + "  " + helpStyle.Render(accountNumber)
	subtitle := "Recent activity"
	if d.account.IBAN != "" {
		subtitle = formatIBAN(d.account.IBAN)
	}
	summary := lipgloss.NewStyle().Foreground(inColor).Render("↓ "+incoming.StringFixed(2)+" "+currency) + "    " + lipgloss.NewStyle().Foreground(outColor).Render("↑ "+outgoing.StringFixed(2)+" "+currency)
	divider := lipgloss.NewStyle().Foreground(mutedColor).Render(strings.Repeat("─", max(width, 1)))
	return heading + "\n" + helpStyle.Render(subtitle) + "\n" + summary + "\n" + divider + "\n"
}
