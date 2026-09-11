package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

func TestMissingAccountStartsWithPickerAndEmptyTransactions(t *testing.T) {
	account := model.Account{AccountNumber: "available-account", BankCode: "2010", Currency: "CZK"}
	m := newModelWithServices(context.Background(), model.Account{}, nil, []model.Account{account}, nil, nil, nil, nil)

	if _, ok := m.current.(*accountPickerScreen); !ok {
		t.Fatal("missing account did not open the account picker")
	}
	if len(m.transactions.list.Items()) != 0 {
		t.Fatalf("missing account loaded transactions: %d", len(m.transactions.list.Items()))
	}
	if view := m.transactions.View(); !strings.Contains(view, "No account") {
		t.Fatalf("empty transactions screen does not show the missing account state:\n%s", view)
	}
	if view := m.View(); !strings.Contains(view, "available-account/2010") {
		t.Fatalf("account picker does not show available account:\n%s", view)
	}
}

func TestAccountShortcutOpensPickerAndEscapeReturns(t *testing.T) {
	account := model.Account{AccountNumber: "first-account", BankCode: "2010", Currency: "CZK"}
	m := newModelWithAccounts(context.Background(), account, nil, []model.Account{account}, nil)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(tuiModel)
	if _, ok := m.current.(*accountPickerScreen); !ok {
		t.Fatal("a did not open the account picker")
	}
	if view := m.View(); !strings.Contains(view, "Switch account") || !strings.Contains(view, "first-account/2010") {
		t.Fatalf("account picker does not show the configured account:\n%s", view)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tuiModel)
	if _, ok := m.current.(*transactionsScreen); !ok {
		t.Fatal("escape did not close the account picker")
	}
}

func TestAccountPickerSwitchesAccount(t *testing.T) {
	current := model.Account{AccountNumber: "first-account", BankCode: "2010", Currency: "CZK"}
	next := model.Account{AccountNumber: "second-account", BankCode: "2010", Currency: "EUR"}
	switcher := func(_ context.Context, accountNumber string) (model.Account, []model.Transaction, error) {
		if accountNumber != next.AccountNumber {
			t.Fatalf("switch requested account %q, want %q", accountNumber, next.AccountNumber)
		}
		return next, nil, nil
	}
	m := newModelWithAccounts(context.Background(), current, nil, []model.Account{current, next}, switcher)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(tuiModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(tuiModel)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	if cmd == nil {
		t.Fatal("selecting an account did not start the switch")
	}
	updated, _ = m.Update(cmd())
	m = updated.(tuiModel)

	if _, ok := m.current.(*transactionsScreen); !ok {
		t.Fatal("successful switch did not return to transactions")
	}
	if view := m.View(); !strings.Contains(view, "second-account/2010") || !strings.Contains(view, "EUR") {
		t.Fatalf("switched view does not show the selected account:\n%s", view)
	}
}

func TestAccountPickerShowsSwitchError(t *testing.T) {
	account := model.Account{AccountNumber: "first-account"}
	picker := newAccountPickerScreen(nil, []model.Account{account}, account.AccountNumber, func(context.Context, string) (model.Account, []model.Transaction, error) {
		return model.Account{}, nil, context.DeadlineExceeded
	}, nil, nil, context.Background(), 80, 24)

	_, cmd, _ := picker.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, _, navigation := picker.Update(cmd())
	if navigation.destination != stay {
		t.Fatal("failed switch navigated away from the picker")
	}
	if view := updated.View(); !strings.Contains(view, "failed switching account") {
		t.Fatalf("picker does not display the switch error:\n%s", view)
	}
}

func TestAccountPickerAddOpensMaskedAPIKeyFormAndRegisters(t *testing.T) {
	current := model.Account{AccountNumber: "first-account", Currency: "CZK"}
	added := model.Account{AccountNumber: "added-account", Currency: "EUR"}
	receivedKey := ""
	registrar := func(_ context.Context, apiKey string) ([]model.Account, model.Account, error) {
		receivedKey = apiKey
		return []model.Account{current, added}, added, nil
	}
	m := newModelWithServices(context.Background(), current, nil, []model.Account{current}, nil, nil, nil, registrar)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(tuiModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(tuiModel)
	if _, ok := m.current.(*addAccountScreen); !ok {
		t.Fatal("a in the account picker did not open the API key form")
	}

	const apiKey = "secret-api-key"
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(apiKey)})
	m = updated.(tuiModel)
	if view := m.View(); strings.Contains(view, apiKey) {
		t.Fatalf("API key is visible in the form:\n%s", view)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	if cmd == nil {
		t.Fatal("submitting the API key did not start registration")
	}
	updated, _ = m.Update(cmd())
	m = updated.(tuiModel)
	if receivedKey != apiKey {
		t.Fatalf("registrar received %q, want the entered API key", receivedKey)
	}
	if _, ok := m.current.(*accountPickerScreen); !ok {
		t.Fatal("successful registration did not return to the account picker")
	}
	if view := m.View(); !strings.Contains(view, "added-account") {
		t.Fatalf("registered account is not shown in the picker:\n%s", view)
	}
}

func TestQIsHandledByAPIKeyForm(t *testing.T) {
	account := model.Account{AccountNumber: "first-account"}
	m := newModelWithServices(context.Background(), account, nil, []model.Account{account}, nil, nil, nil, nil)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(tuiModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(tuiModel)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(tuiModel)
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("q quit while entering the API key")
		}
	}
	if key := m.current.(*addAccountScreen).form.input.Value(); key != "q" {
		t.Fatalf("API key input is %q, want q", key)
	}
}

func TestDeleteOpensConfirmationAndCanBeCancelled(t *testing.T) {
	account := model.Account{AccountNumber: "first-account", BankCode: "2010"}
	m := newModelWithServices(context.Background(), account, nil, []model.Account{account}, nil, nil, nil, nil)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(tuiModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(tuiModel)
	if _, ok := m.current.(*removeAccountConfirmationScreen); !ok {
		t.Fatal("d did not open the removal confirmation")
	}
	for _, want := range []string{"Remove account?", "first-account/2010", "y/enter confirm"} {
		if view := m.View(); !strings.Contains(view, want) {
			t.Fatalf("confirmation does not contain %q:\n%s", want, view)
		}
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tuiModel)
	if _, ok := m.current.(*accountPickerScreen); !ok {
		t.Fatal("escape did not return to the account picker")
	}
}

func TestConfirmedDeleteRemovesAccount(t *testing.T) {
	account := model.Account{AccountNumber: "first-account"}
	removed := false
	remover := func(context.Context, string) (AccountState, error) {
		removed = true
		return AccountState{}, nil
	}
	m := newModelWithServices(context.Background(), account, nil, []model.Account{account}, nil, nil, remover, nil)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(tuiModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(tuiModel)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(tuiModel)
	if cmd == nil {
		t.Fatal("confirming removal did not start the operation")
	}
	updated, _ = m.Update(cmd())
	m = updated.(tuiModel)
	if !removed {
		t.Fatal("confirmed removal did not call the remover")
	}
}

func TestShiftDeleteSkipsConfirmation(t *testing.T) {
	account := model.Account{AccountNumber: "first-account"}
	removed := ""
	remover := func(_ context.Context, accountNumber string) (AccountState, error) {
		removed = accountNumber
		return AccountState{}, nil
	}
	m := newModelWithServices(context.Background(), account, nil, []model.Account{account}, nil, nil, remover, nil)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(tuiModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(tuiModel)
	if view := m.View(); !strings.Contains(view, "Shift+D in the account picker") {
		t.Fatalf("confirmation does not explain the Shift+D shortcut:\n%s", view)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tuiModel)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	m = updated.(tuiModel)
	if _, ok := m.current.(*accountPickerScreen); !ok || cmd == nil {
		t.Fatal("Shift+D did not start immediate removal from the picker")
	}
	updated, _ = m.Update(cmd())
	m = updated.(tuiModel)
	if removed != account.AccountNumber {
		t.Fatalf("removed account %q, want %q", removed, account.AccountNumber)
	}
	if _, ok := m.current.(*transactionsScreen); !ok {
		t.Fatal("successful removal did not return to transactions")
	}
}
