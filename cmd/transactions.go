package cmd

import (
	"fmt"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	transactionstui "go.chrastecky.dev/fio/fiocli/cmd/tui/transactions"
)

var transactionsCmd = &cobra.Command{
	Use:   "transactions",
	Short: "Lists your current account's transactions",
	RunE: func(cmd *cobra.Command, args []string) error {
		account, err := client.Account(cmd.Context(), viper.GetString("current-account"))
		if err != nil {
			return fmt.Errorf("failed fetching current account: %w", err)
		}

		if lo.Must(cmd.Flags().GetBool("sync")) {
			if _, err := account.LoadNewTransactions(cmd.Context()); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "failed fetching new transactions")
			}
		}

		transactions, err := account.Transactions(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed getting transactions: %w", err)
		}

		nonInteractive := lo.Must(cmd.Flags().GetBool("non-interactive"))
		if nonInteractive {
			limit := lo.Must(cmd.Flags().GetInt("limit"))
			if lo.Must(cmd.Flags().GetBool("json")) {
				if err := transactionstui.RenderJSON(cmd.OutOrStdout(), transactions, limit); err != nil {
					return err
				}
				return nil
			}

			transactionstui.RenderTable(cmd.OutOrStdout(), transactions, limit)
			return nil
		}

		if err := transactionstui.Run(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), transactions); err != nil {
			return fmt.Errorf("failed rendering transactions: %w", err)
		}

		return nil
	},
}

func init() {
	transactionsCmd.Flags().Bool("sync", false, "Attempt sync before listing them")
	transactionsCmd.Flags().Bool("non-interactive", false, "Non-interactive mode")
	transactionsCmd.Flags().Int("limit", 20, "Limit the number of transactions to return (only for non-interactive mode)")
	transactionsCmd.Flags().Bool("json", false, "Output results as JSON (only for non-interactive mode)")

	rootCmd.AddCommand(transactionsCmd)
}
