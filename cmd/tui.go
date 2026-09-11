package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.chrastecky.dev/fio-api/fio/dto"
	"go.chrastecky.dev/fio-client/fioclient/model"
	"go.chrastecky.dev/fio/fiocli/cmd/helper"
	"go.chrastecky.dev/fio/fiocli/cmd/tui"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Open the interactive account transaction browser",
	Aliases: []string{
		"tui",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		loadState := func(ctx context.Context) (tui.AccountState, error) {
			if client == nil {
				return tui.AccountState{}, errors.New("fio client is unavailable")
			}
			accounts, err := client.Accounts(ctx)
			if err != nil {
				return tui.AccountState{}, fmt.Errorf("failed listing accounts: %w", err)
			}
			state := tui.AccountState{Accounts: accounts}
			account, err := client.Account(ctx, viper.GetString("current-account"))
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return tui.AccountState{}, fmt.Errorf("failed fetching current account: %w", err)
			}
			if account == nil {
				return state, nil
			}
			state.Account = account.AccountData()
			state.Transactions, err = account.Transactions(ctx)
			if err != nil {
				return tui.AccountState{}, fmt.Errorf("failed getting transactions: %w", err)
			}
			return state, nil
		}

		locked := viper.GetString("encryption-password") == ""
		var state tui.AccountState
		if !locked {
			var err error
			state, err = loadState(cmd.Context())
			locked = err != nil
		}

		switchAccount := func(ctx context.Context, accountNumber string) (model.Account, []model.Transaction, error) {
			selected, err := client.Account(ctx, accountNumber)
			if err != nil {
				return model.Account{}, nil, fmt.Errorf("failed getting account: %w", err)
			}
			transactions, err := selected.Transactions(ctx)
			if err != nil {
				return model.Account{}, nil, fmt.Errorf("failed getting transactions: %w", err)
			}
			if err := helper.SetCurrentAccount(cfgFile, accountNumber); err != nil {
				return model.Account{}, nil, err
			}
			viper.Set("current-account", accountNumber)
			return selected.AccountData(), transactions, nil
		}
		reloadTransactions := func(ctx context.Context, accountNumber string) ([]model.Transaction, error) {
			selected, err := client.Account(ctx, accountNumber)
			if err != nil {
				return nil, fmt.Errorf("failed getting account: %w", err)
			}
			if _, err := selected.LoadNewTransactions(ctx); err != nil {
				return nil, fmt.Errorf("failed fetching new transactions: %w", err)
			}
			return selected.Transactions(ctx)
		}
		loadTransactions := func(ctx context.Context, accountNumber string) ([]model.Transaction, error) {
			selected, err := client.Account(ctx, accountNumber)
			if err != nil {
				return nil, fmt.Errorf("failed getting account: %w", err)
			}
			return selected.Transactions(ctx)
		}
		removeAccount := func(ctx context.Context, accountNumber string) (tui.AccountState, error) {
			if err := client.RemoveAccount(ctx, accountNumber); err != nil {
				return tui.AccountState{}, err
			}
			remaining, err := client.Accounts(ctx)
			if err != nil {
				return tui.AccountState{}, fmt.Errorf("failed listing remaining accounts: %w", err)
			}

			activeNumber := viper.GetString("current-account")
			if activeNumber == accountNumber {
				activeNumber = ""
				if len(remaining) > 0 {
					activeNumber = remaining[0].AccountNumber
				}
				if err := helper.SetCurrentAccount(cfgFile, activeNumber); err != nil {
					return tui.AccountState{}, err
				}
				viper.Set("current-account", activeNumber)
			}

			state := tui.AccountState{Accounts: remaining}
			if activeNumber == "" {
				return state, nil
			}
			active, err := client.Account(ctx, activeNumber)
			if err != nil {
				return tui.AccountState{}, fmt.Errorf("failed getting active account: %w", err)
			}
			state.Account = active.AccountData()
			state.Transactions, err = active.Transactions(ctx)
			if err != nil {
				return tui.AccountState{}, fmt.Errorf("failed getting active transactions: %w", err)
			}
			return state, nil
		}
		registerAccount := func(ctx context.Context, apiKey string) ([]model.Account, model.Account, error) {
			added, err := client.RegisterAccount(ctx, apiKey, false)
			if err != nil {
				return nil, model.Account{}, fmt.Errorf("is the API key correct? %w", err)
			}
			accounts, err := client.Accounts(ctx)
			if err != nil {
				return nil, model.Account{}, fmt.Errorf("failed listing accounts: %w", err)
			}
			return accounts, added, nil
		}
		createPayment := func(ctx context.Context, accountNumber string, payment dto.DomesticTransaction) error {
			account, err := client.Account(ctx, accountNumber)
			if err != nil {
				return fmt.Errorf("failed getting account: %w", err)
			}
			if _, err := account.CreateDomesticPayment(ctx, payment); err != nil {
				return fmt.Errorf("failed creating payment: %w", err)
			}
			return nil
		}

		unlockDatabase := func(ctx context.Context, password string) (tui.AccountState, error) {
			// Recreate the client: it may have been opened with an empty or incorrect key.
			client = nil
			viper.Set("encryption-password", password)
			initializeDb(false, cmd.ErrOrStderr())
			if client == nil {
				return tui.AccountState{}, errors.New("failed creating fio client")
			}
			return loadState(ctx)
		}
		services := tui.Services{SwitchAccount: switchAccount, ReloadTransactions: reloadTransactions, LoadTransactions: loadTransactions, RemoveAccount: removeAccount, RegisterAccount: registerAccount, UnlockDatabase: unlockDatabase, CreatePayment: createPayment}
		if err := tui.Run(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), state, services, locked); err != nil {
			return fmt.Errorf("failed rendering transactions: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
