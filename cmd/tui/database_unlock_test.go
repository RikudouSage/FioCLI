package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

func TestLockedDatabaseStartsOnUnlockScreen(t *testing.T) {
	m := newModelWithState(context.Background(), AccountState{}, Services{UnlockDatabase: func(context.Context, string) (AccountState, error) {
		return AccountState{}, nil
	}}, true)
	if _, ok := m.current.(*databaseUnlockScreen); !ok {
		t.Fatal("locked database did not open the unlock screen")
	}
}

func TestUnlockTransitionsToLoadedAccount(t *testing.T) {
	account := model.Account{AccountNumber: "123456", BankCode: "2010", Currency: "CZK"}
	m := newModelWithState(context.Background(), AccountState{}, Services{UnlockDatabase: func(_ context.Context, password string) (AccountState, error) {
		if password != "correct password" {
			t.Fatalf("unlock received password %q", password)
		}
		return AccountState{Account: account, Accounts: []model.Account{account}}, nil
	}}, true)

	screen := m.current.(*databaseUnlockScreen)
	screen.input.SetValue("correct password")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	if cmd == nil {
		t.Fatal("unlock did not invoke the database service")
	}
	updated, _ = m.Update(cmd())
	m = updated.(tuiModel)
	if _, ok := m.current.(*transactionsScreen); !ok {
		t.Fatal("successful unlock did not open transactions")
	}
	if !strings.Contains(m.View(), "123456/2010") {
		t.Fatalf("unlocked account is not displayed:\n%s", m.View())
	}
}

func TestUnlockErrorStaysOnScreen(t *testing.T) {
	m := newModelWithState(context.Background(), AccountState{}, Services{UnlockDatabase: func(context.Context, string) (AccountState, error) {
		return AccountState{}, errors.New("invalid password")
	}}, true)
	screen := m.current.(*databaseUnlockScreen)
	screen.input.SetValue("wrong password")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	updated, _ = m.Update(cmd())
	m = updated.(tuiModel)
	if _, ok := m.current.(*databaseUnlockScreen); !ok {
		t.Fatal("failed unlock left the unlock screen")
	}
	if !strings.Contains(m.View(), "invalid password") {
		t.Fatalf("unlock error is not displayed:\n%s", m.View())
	}
}

func TestQIsHandledByUnlockPasswordInput(t *testing.T) {
	m := newModelWithState(context.Background(), AccountState{}, Services{}, true)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(tuiModel)
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("q quit while entering the database password")
		}
	}
	screen := m.current.(*databaseUnlockScreen)
	if screen.input.Value() != "q" {
		t.Fatalf("password input is %q, want q", screen.input.Value())
	}
}
