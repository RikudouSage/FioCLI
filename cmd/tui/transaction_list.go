package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/shopspring/decimal"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

type transactionDelegate struct {
	normalTitle, normalDesc, selectedTitle, selectedDesc lipgloss.Style
}

func newTransactionDelegate() transactionDelegate {
	return transactionDelegate{
		normalTitle: lipgloss.NewStyle().PaddingLeft(2), normalDesc: lipgloss.NewStyle().PaddingLeft(2),
		selectedTitle: lipgloss.NewStyle().Border(lipgloss.ThickBorder(), false, false, false, true).BorderForeground(primaryColor).PaddingLeft(1).Bold(true),
		selectedDesc:  lipgloss.NewStyle().Border(lipgloss.ThickBorder(), false, false, false, true).BorderForeground(primaryColor).PaddingLeft(1),
	}
}

// Most rows use three terminal lines including the list separator. Reserve one
// additional line so a page beginning with the original multi-line month
// summary stays within the list viewport instead of pushing the dashboard off
// the top of the terminal.
func (transactionDelegate) Height() int                         { return 4 }
func (transactionDelegate) Spacing() int                        { return 0 }
func (transactionDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d transactionDelegate) Render(writer io.Writer, transactionList list.Model, index int, raw list.Item) {
	item, ok := raw.(transactionItem)
	if !ok || transactionList.Width() <= 0 {
		return
	}
	titleStyle, descriptionStyle := d.normalTitle, d.normalDesc
	if index == transactionList.Index() && transactionList.FilterState() != list.Filtering {
		titleStyle, descriptionStyle = d.selectedTitle, d.selectedDesc
	}
	pending := item.transaction.LocalOnly
	titleStyle, descriptionStyle = withPendingStyle(titleStyle, pending), withPendingStyle(descriptionStyle, pending)
	contentWidth := max(transactionList.Width()-titleStyle.GetHorizontalFrameSize(), 1)
	monthHeader := ""
	if isMonthStart(transactionList, index, item.transaction) {
		incoming, outgoing, currency := monthTotals(transactionList, item.transaction)
		monthHeader = renderMonthHeader(item.transaction.Date.AsTime().Format("January 2006"), incoming, outgoing, currency, transactionList.Width())
	}
	amount := formatAmount(item.transaction)
	name := ansi.Truncate(item.displayName(), max(contentWidth-lipgloss.Width(amount)-2, 1), "…")
	space := strings.Repeat(" ", max(contentWidth-lipgloss.Width(name)-lipgloss.Width(amount), 1))
	amountStyle := lipgloss.NewStyle().Foreground(outColor).Italic(pending)
	if item.transaction.Amount.IsPositive() {
		amountStyle = amountStyle.Foreground(inColor)
	}
	title := name + space + amountStyle.Render(amount)
	metadata := lipgloss.NewStyle().Foreground(mutedColor).Italic(pending).Render(item.metadata())
	if paymentType := item.paymentType(); paymentType != "" {
		metadata += lipgloss.NewStyle().Foreground(faintColor).Italic(pending).Render("  •  " + paymentType)
	}
	description := metadata
	if comment := item.comment(); comment != "" {
		description = lipgloss.NewStyle().Bold(true).Italic(pending).Foreground(primaryColor).Render(comment) + lipgloss.NewStyle().Foreground(mutedColor).Italic(pending).Render("  •  ") + metadata
	}
	if pending {
		description = lipgloss.NewStyle().Bold(true).Italic(true).Foreground(pendingColor).Render("Pending confirmation") + lipgloss.NewStyle().Foreground(mutedColor).Italic(true).Render("  •  ") + description
	}
	description = ansi.Truncate(description, contentWidth, "…")
	if monthHeader != "" {
		fmt.Fprintf(writer, "%s\n%s\n%s", monthHeader, titleStyle.Render(title), descriptionStyle.Render(description))
	} else {
		fmt.Fprintf(writer, "\n%s\n%s", titleStyle.Render(title), descriptionStyle.Render(description))
	}
	if index == len(transactionList.VisibleItems())-1 {
		fmt.Fprintf(writer, "\n%s", endOfTransactions(transactionList.Width()))
	}
}

func endOfTransactions(width int) string {
	return lipgloss.NewStyle().Foreground(faintColor).Render(lipgloss.PlaceHorizontal(max(width, 1), lipgloss.Center, "No more transactions"))
}

func renderMonthHeader(month, incoming, outgoing, currency string, width int) string {
	indent := "  "
	title := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(strings.ToUpper(month))
	label := lipgloss.NewStyle().Foreground(mutedColor)
	stats := label.Render("Incoming ") + lipgloss.NewStyle().Bold(true).Foreground(inColor).Render("+"+incoming+" "+currency) + label.Render("    Outgoing ") + lipgloss.NewStyle().Bold(true).Foreground(outColor).Render("−"+outgoing+" "+currency)
	divider := lipgloss.NewStyle().Foreground(faintColor).Render(strings.Repeat("─", max(width-len(indent), 1)))
	return "\n" + indent + title + "\n" + indent + stats + "\n" + indent + divider
}

func monthTotals(transactionList list.Model, transaction model.Transaction) (string, string, string) {
	month, currency := transaction.Date.AsTime(), transaction.Currency
	incoming, outgoing := decimal.Zero, decimal.Zero
	for _, raw := range transactionList.Items() {
		candidate, ok := raw.(transactionItem)
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
