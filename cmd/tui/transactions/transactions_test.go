package transactions

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shopspring/decimal"
	"go.chrastecky.dev/fio-api/fio/types"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

func TestEnterOpensDetailAndEscapeReturnsToList(t *testing.T) {
	m := newModel(testAccount(), []model.Transaction{testTransaction(t)})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	if m.screen != detailScreen {
		t.Fatal("enter did not open the detail screen")
	}
	view := m.View()
	for _, want := range []string{"Transaction details", "Coffee Shop", "-125.50 CZK", "123456/2010"} {
		if !strings.Contains(view, want) {
			t.Errorf("detail view does not contain %q", want)
		}
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tuiModel)
	if m.screen != listScreen {
		t.Fatal("escape did not return to the list screen")
	}
}

func TestEmptyListIgnoresEnter(t *testing.T) {
	m := newModel(testAccount(), nil)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	if m.screen != listScreen {
		t.Fatal("empty list opened the detail screen")
	}
}

func TestFilteredListDoesNotRenderBrokenEscapeSequences(t *testing.T) {
	transaction := testTransaction(t)
	transaction.CounterpartyAccount = "example-8898-account"
	m := newModel(testAccount(), []model.Transaction{transaction})
	m.list.SetFilterText("8898")

	view := m.View()
	if strings.Contains(view, "[0m[") || strings.Contains(view, "[38;5;") {
		t.Fatalf("filtered view contains a broken ANSI escape sequence:\n%s", view)
	}
	if strings.Contains(view, "\x1b[4m") {
		t.Fatalf("filtered metadata match was incorrectly highlighted in the title:\n%s", view)
	}
	if !strings.Contains(view, "-125.50 CZK") || !strings.Contains(view, "example-8898-account/2010") {
		t.Fatalf("filtered view does not contain the transaction data:\n%s", view)
	}
}

func TestTransactionCanBeFilteredByAmount(t *testing.T) {
	item := transactionItem{transaction: testTransaction(t)}
	filterValue := item.FilterValue()

	for _, want := range []string{"-125.50 CZK", "125.50"} {
		if !strings.Contains(filterValue, want) {
			t.Errorf("filter value %q does not contain amount %q", filterValue, want)
		}
	}
}

func TestSubstringFilterDoesNotFuzzyMatchAccountNumber(t *testing.T) {
	targets := []string{
		"-125.00 CZK 1325090010/3030",
		"100.00 CZK 1325090010/3030",
		"25.00 CZK example account",
	}

	ranks := substringFilter("100", targets)
	if len(ranks) != 1 || ranks[0].Index != 1 {
		t.Fatalf("substringFilter returned %#v, want only index 1", ranks)
	}
}

func TestResizeUpdatesBothScreens(t *testing.T) {
	m := newModel(testAccount(), []model.Transaction{testTransaction(t)})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(tuiModel)
	if m.list.Width() != 100 || m.list.Height() != 25 {
		t.Fatalf("list size = %dx%d, want 100x25", m.list.Width(), m.list.Height())
	}
	if m.viewport.Width != 96 || m.viewport.Height != 23 {
		t.Fatalf("viewport size = %dx%d, want 96x23", m.viewport.Width, m.viewport.Height)
	}
}

func TestDashboardUsesAccountData(t *testing.T) {
	m := newModel(testAccount(), []model.Transaction{testTransaction(t)})
	view := m.View()
	for _, want := range []string{"example-account/2010", "CZ00 EXAM PLEI BAN"} {
		if !strings.Contains(view, want) {
			t.Errorf("dashboard does not contain account data %q", want)
		}
	}
}

func TestFormatIBAN(t *testing.T) {
	for input, want := range map[string]string{
		"CZ6508000000192000145399":     "CZ65 0800 0000 1920 0014 5399",
		"CZ65 0800 00001920 0014 5399": "CZ65 0800 0000 1920 0014 5399",
		"":                             "",
	} {
		if got := formatIBAN(input); got != want {
			t.Errorf("formatIBAN(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTranslateTransactionType(t *testing.T) {
	for input, want := range map[string]string{
		"Platba kartou":           "Card payment",
		"Okamžitá odchozí platba": "Instant outgoing payment",
		"Future API value":        "Future API value",
	} {
		if got := translateTransactionType(input); got != want {
			t.Errorf("translateTransactionType(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRenderTableIncludesAllTransactionDetails(t *testing.T) {
	transaction := testTransaction(t)
	constantSymbol := "0308"
	variableSymbol := "1234567890"
	specificSymbol := "42"
	userIdentity := "Invoice payment"
	performedBy := "Card holder"
	additionalInfo := "Terminal 123"
	bic := "FIOBCZPPXXX"
	instructionID := int64(987)
	payerReference := "REF-2026"
	transaction.ConstantSymbol = &constantSymbol
	transaction.VariableSymbol = &variableSymbol
	transaction.SpecificSymbol = &specificSymbol
	transaction.UserIdentity = &userIdentity
	transaction.PerformedBy = &performedBy
	transaction.AdditionalInfo = &additionalInfo
	transaction.BIC = &bic
	transaction.InstructionID = &instructionID
	transaction.PayerReference = &payerReference

	var output bytes.Buffer
	RenderTable(&output, []model.Transaction{transaction}, 20)

	for _, want := range []string{
		"Account number", "Counterparty bank code", "Constant symbol", "Variable symbol",
		"Specific symbol", "User identity", "Performed by", "Additional info", "Comment",
		"BIC", "Instruction ID", "Payer reference", "9876543210", "Morning coffee",
		"REF-2026",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("table output does not contain %q", want)
		}
	}
}

func TestRenderJSONIncludesDetailsAndHonorsLimit(t *testing.T) {
	first := testTransaction(t)
	second := testTransaction(t)
	second.ID = 43

	var output bytes.Buffer
	if err := RenderJSON(&output, []model.Transaction{first, second}, 1); err != nil {
		t.Fatal(err)
	}

	var decoded []map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("rendered invalid JSON: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("rendered %d transactions, want 1", len(decoded))
	}
	for key, want := range map[string]any{
		"id":                     float64(42),
		"account_number":         "9876543210",
		"counterparty_name":      "Coffee Shop",
		"counterparty_bank_code": "2010",
		"comment":                "Morning coffee",
	} {
		if got := decoded[0][key]; got != want {
			t.Errorf("%s = %#v, want %#v", key, got, want)
		}
	}
}

func testTransaction(t *testing.T) model.Transaction {
	t.Helper()
	var transactionType types.TransactionType
	if err := transactionType.UnmarshalText([]byte("Platba kartou")); err != nil {
		t.Fatal(err)
	}
	comment := "Morning coffee"
	return model.Transaction{
		ID:                   42,
		AccountNumber:        "9876543210",
		Date:                 types.TimezonedDate(time.Date(2026, time.September, 11, 0, 0, 0, 0, time.FixedZone("CEST", 2*60*60))),
		Amount:               decimal.RequireFromString("-125.50"),
		Currency:             "CZK",
		CounterpartyAccount:  "123456",
		CounterpartyBankCode: "2010",
		CounterpartyName:     "Coffee Shop",
		CounterpartyBankName: "Fio banka",
		TransactionType:      transactionType,
		Comment:              &comment,
	}
}

func testAccount() model.Account {
	return model.Account{
		AccountNumber: "example-account",
		BankCode:      "2010",
		Currency:      "CZK",
		IBAN:          "CZ00 EXAMPLE IBAN",
		BIC:           "EXAMPLEBIC",
	}
}
