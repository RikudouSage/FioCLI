package transactions

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/fatih/color"
	"github.com/shopspring/decimal"
	"go.chrastecky.dev/fio-api/fio/types"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

func TestTranslateTransactionType(t *testing.T) {
	for input, want := range map[string]string{
		"Platba kartou":           "Card payment",
		"Okamžitá odchozí platba": "Instant outgoing payment",
		"Future API value":        "Future API value",
	} {
		if got := TranslateTransactionType(input); got != want {
			t.Errorf("TranslateTransactionType(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRenderTableIncludesSelectedTransactionDetails(t *testing.T) {
	transaction := outputTestTransaction(t)
	var output bytes.Buffer
	RenderTable(&output, []model.Transaction{transaction}, 20)
	for _, want := range []string{"ID", "Date", "Amount", "Counterparty", "Transaction type", "Local only", "Coffee Shop", "Morning coffee"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("table output does not contain %q", want)
		}
	}
}

func TestRenderJSONIncludesDetailsAndHonorsLimit(t *testing.T) {
	first := outputTestTransaction(t)
	second := outputTestTransaction(t)
	second.ID = 43
	var output bytes.Buffer
	if err := RenderJSON(&output, []model.Transaction{first, second}, 1); err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("rendered invalid JSON: %v", err)
	}
	if len(decoded) != 1 || decoded[0]["id"] != float64(42) {
		t.Fatalf("unexpected JSON output: %#v", decoded)
	}
}

func TestRenderDetailFormatsCompleteTransaction(t *testing.T) {
	transaction := outputTestTransaction(t)
	transaction.Amount = decimal.RequireFromString("-1234567.5")
	transaction.LocalOnly = true
	variableSymbol := "1234567890"
	transaction.VariableSymbol = &variableSymbol
	account := model.Account{
		AccountNumber: "example-account",
		BankCode:      "2010",
		IBAN:          "CZ6508000000192000145399",
	}

	var output bytes.Buffer
	RenderDetail(&output, account, transaction)
	plain := ansi.Strip(output.String())
	for _, want := range []string{
		"Transaction ID", "42",
		"Pending confirmation",
		"example-account/2010",
		"CZ65 0800 0000 1920 0014 5399",
		"11 September 2026",
		"-1,234,567.50 CZK",
		"Card payment",
		"Variable symbol", "1234567890",
		"Constant symbol", "—",
		"Morning coffee",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("detail output does not contain %q:\n%s", want, plain)
		}
	}
}

func TestRenderTransactionJSONWritesObject(t *testing.T) {
	var output bytes.Buffer
	if err := RenderTransactionJSON(&output, outputTestTransaction(t)); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("rendered invalid JSON: %v", err)
	}
	if decoded["id"] != float64(42) {
		t.Fatalf("unexpected JSON output: %#v", decoded)
	}
}

func TestRenderTableAlignsColoredColumns(t *testing.T) {
	previousNoColor := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = previousNoColor })

	transaction := outputTestTransaction(t)
	payerReference := "EXAMPLE-REFERENCE"
	transaction.PayerReference = &payerReference
	var output bytes.Buffer
	RenderTable(&output, []model.Transaction{transaction}, 20)
	lines := strings.Split(strings.TrimSpace(ansi.Strip(output.String())), "\n")
	if len(lines) < 2 {
		t.Fatalf("table output has %d lines, want at least 2", len(lines))
	}

	columns := [][2]string{
		{"Transaction type", "Card payment"},
		{"Comment", "Morning coffee"},
		{"Payer reference", "EXAMPLE-REFERENCE"},
		{"Local only", "✗"},
	}
	for _, column := range columns {
		headerColumn := strings.Index(lines[0], column[0])
		rowColumn := strings.Index(lines[1], column[1])
		if headerColumn < 0 || rowColumn < 0 || headerColumn != rowColumn {
			t.Errorf("%s column is misaligned: header=%d row=%d\n%s", column[0], headerColumn, rowColumn, ansi.Strip(output.String()))
		}
	}
}

func outputTestTransaction(t *testing.T) model.Transaction {
	t.Helper()
	var transactionType types.TransactionType
	if err := transactionType.UnmarshalText([]byte("Platba kartou")); err != nil {
		t.Fatal(err)
	}
	comment := "Morning coffee"
	return model.Transaction{
		ID: 42, AccountNumber: "example-account", Date: types.TimezonedDate(time.Date(2026, time.September, 11, 0, 0, 0, 0, time.UTC)),
		Amount: decimal.RequireFromString("-125.50"), Currency: "CZK", CounterpartyAccount: "example-counterparty",
		CounterpartyBankCode: "2010", CounterpartyName: "Coffee Shop", TransactionType: transactionType, Comment: &comment,
	}
}
