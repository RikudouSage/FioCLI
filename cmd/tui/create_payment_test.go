package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shopspring/decimal"
	"go.chrastecky.dev/fio-api/fio/dto"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

func TestCreatePaymentFormOrdersFieldsByFrequency(t *testing.T) {
	screen := newCreatePaymentScreen(nil, model.Account{Currency: "CZK"}, nil, context.Background(), 80, 40)
	if got, want := screen.fields[paymentAccountTo].section, "Required"; got != want {
		t.Fatalf("first field section = %q, want %q", got, want)
	}
	if got, want := screen.fields[paymentCurrency].section, "Common"; got != want {
		t.Fatalf("currency section = %q, want %q", got, want)
	}
	if got, want := screen.fields[paymentDate].section, "Advanced"; got != want {
		t.Fatalf("date section = %q, want %q", got, want)
	}
	view := screen.View()
	if required, common, advanced := strings.Index(view, "REQUIRED FIELDS"), strings.Index(view, "COMMON FIELDS"), strings.Index(view, "ADVANCED FIELDS"); required < 0 || common < required || advanced < common {
		t.Fatalf("sections are not rendered in required/common/advanced order:\n%s", view)
	}
}

func TestCreatePaymentFormSubmitsDomesticPayment(t *testing.T) {
	var source string
	var created dto.DomesticTransaction
	screen := newCreatePaymentScreen(nil, model.Account{AccountNumber: "source", Currency: "CZK"}, func(_ context.Context, account string, payment dto.DomesticTransaction) error {
		source, created = account, payment
		return nil
	}, context.Background(), 80, 40)
	screen.fields[paymentAccountTo].input.SetValue("123456789")
	screen.fields[paymentBankCode].input.SetValue("2010")
	screen.fields[paymentAmount].input.SetValue("42.50")
	screen.fields[paymentMessage].input.SetValue("Invoice 42")

	_, cmd, _ := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter did not create a payment command")
	}
	result, ok := cmd().(paymentCreationResult)
	if !ok {
		t.Fatalf("payment command result = %T, want paymentCreationResult", cmd())
	}
	if result.err != nil {
		t.Fatalf("payment creation failed: %v", result.err)
	}
	if source != "source" || created.AccountTo != "123456789" || created.BankCode != "2010" || !created.Amount.Equal(decimal.RequireFromString("42.50")) {
		t.Fatalf("unexpected payment: source=%q payment=%+v", source, created)
	}
	if got := pointerString(created.MessageForRecipient); got != "Invoice 42" {
		t.Fatalf("message = %q, want Invoice 42", got)
	}
}

func TestCreatePaymentFormCurrencySelectorUsesFioCurrencies(t *testing.T) {
	screen := newCreatePaymentScreen(nil, model.Account{Currency: "CZK"}, nil, context.Background(), 80, 40)
	screen.focus(paymentCurrency)
	screen.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := screen.fields[paymentCurrency].choices[screen.fields[paymentCurrency].selected]; got != "USD" {
		t.Fatalf("currency after right = %q, want USD", got)
	}
	if view := screen.View(); !strings.Contains(view, "< USD >") {
		t.Fatalf("currency selector is not shown:\n%s", view)
	}
}

func TestCreatePaymentFormPaymentTypeSelector(t *testing.T) {
	screen := newCreatePaymentScreen(nil, model.Account{}, nil, context.Background(), 80, 40)
	screen.focus(paymentType)
	screen.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := screen.fields[paymentType].choices[screen.fields[paymentType].selected]; got != "priority" {
		t.Fatalf("payment type after right = %q, want priority", got)
	}
	if view := screen.View(); !strings.Contains(view, "< priority >") {
		t.Fatalf("payment type selector is not shown:\n%s", view)
	}
}

func TestSuccessfulPaymentRefreshesTransactionList(t *testing.T) {
	transactions := newTransactionsScreenWithReload(model.Account{AccountNumber: "source"}, nil, func(context.Context, string) ([]model.Transaction, error) {
		return nil, nil
	}, context.Background())
	transactions.loadTransactions = func(context.Context, string) ([]model.Transaction, error) { return nil, nil }
	screen := newCreatePaymentScreen(transactions, model.Account{AccountNumber: "source"}, nil, context.Background(), 80, 40)

	updated, cmd, _ := screen.Update(paymentCreationResult{})
	if updated != transactions || cmd == nil {
		t.Fatal("a successful payment did not return to and refresh the transaction list")
	}
	if !transactions.reloading {
		t.Fatal("payment success did not start a transaction refresh")
	}
}

func TestCreatePaymentFormWrapsLongErrors(t *testing.T) {
	screen := newCreatePaymentScreen(nil, model.Account{}, nil, context.Background(), 50, 40)
	screen.err = errors.New("failed creating payment: Fio rejected the request because the recipient account is unavailable")
	view := screen.View()
	if !strings.Contains(view, "recipient") || !strings.Contains(view, "unavailable") {
		t.Fatalf("long error was cut instead of wrapped:\n%s", view)
	}
}
