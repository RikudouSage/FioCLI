package cmd

import (
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

var syncCmd = &cobra.Command{
	Use:   "sync [flags]",
	Short: "Synchronize transactions for the currently selected account",
	RunE: func(cmd *cobra.Command, args []string) error {
		account, err := client.Account(cmd.Context(), viper.GetString("current-account"))
		if err != nil {
			return fmt.Errorf("failed fetching current account: %w", err)
		}

		var fetcher func() ([]model.Transaction, error)

		force := lo.Must(cmd.Flags().GetBool("force"))
		if force {
			fetcher = func() ([]model.Transaction, error) {
				end := time.Now()
				start := time.Now().AddDate(0, 0, -90)
				return account.LoadTransactionsByDate(cmd.Context(), start, end)
			}
		} else {
			fetcher = func() ([]model.Transaction, error) {
				return account.LoadNewTransactions(cmd.Context())
			}
		}

		transactions, err := fetcher()
		if err != nil {
			return fmt.Errorf("failed fetching transactions: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully fetched %d transactions\n", len(transactions))
		return nil
	},
}

func init() {
	syncCmd.Flags().Bool("force", false, "Force sync of the last 90 days instead of only new transactions")

	rootCmd.AddCommand(syncCmd)
}
