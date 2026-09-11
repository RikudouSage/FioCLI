package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shopspring/decimal"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

func TestReloadShortcutRefreshesTransactions(t *testing.T) {
	account := model.Account{AccountNumber: "active-account", Currency: "EUR"}
	fresh := model.Transaction{ID: 2, CounterpartyName: "Fresh transaction", Amount: decimal.NewFromInt(10), Currency: "EUR"}
	requestedAccount := ""
	screen := newTransactionsScreenWithReload(account, nil, func(_ context.Context, accountNumber string) ([]model.Transaction, error) {
		requestedAccount = accountNumber
		return []model.Transaction{fresh}, nil
	}, context.Background())

	_, cmd, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil || !screen.reloading {
		t.Fatal("r did not start a transaction reload")
	}
	if view := screen.View(); !strings.Contains(view, "Syncing transactions…") {
		t.Fatalf("reload indicator is not visible:\n%s", view)
	}

	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatal("reload did not return a command batch")
	}
	for _, batchCommand := range batch {
		if result, ok := batchCommand().(transactionReloadResult); ok {
			screen.Update(result)
		}
	}

	if requestedAccount != account.AccountNumber {
		t.Fatalf("reload requested account %q, want %q", requestedAccount, account.AccountNumber)
	}
	if screen.reloading {
		t.Fatal("screen remained in the reloading state")
	}
	if view := screen.View(); strings.Contains(view, "Syncing transactions…") {
		t.Fatalf("reload indicator remained visible after completion:\n%s", view)
	}
	if view := screen.View(); !strings.Contains(view, "Fresh transaction") {
		t.Fatalf("reloaded transaction is not displayed:\n%s", view)
	}
}

func TestReloadShortcutIsIgnoredWhileFiltering(t *testing.T) {
	called := false
	screen := newTransactionsScreenWithReload(model.Account{}, nil, func(context.Context, string) ([]model.Transaction, error) {
		called = true
		return nil, nil
	}, context.Background())
	screen.list.SetFilterState(list.Filtering)
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if called || screen.reloading {
		t.Fatal("r triggered a reload while editing the filter")
	}
}
