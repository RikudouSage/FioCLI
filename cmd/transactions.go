package cmd

import (
	"fmt"
	"strconv"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.chrastecky.dev/fio-client/fioclient/account"
	transactionoutput "go.chrastecky.dev/fio/fiocli/cmd/transactions"
)

func displaySingle(cmd *cobra.Command, id int64, account account.Account) error {
	transaction, err := account.Transaction(cmd.Context(), id)
	if err != nil {
		return fmt.Errorf("failed fetching transaction: %w", err)
	}

	if lo.Must(cmd.Flags().GetBool("json")) {
		return transactionoutput.RenderTransactionJSON(cmd.OutOrStdout(), transaction)
	}

	transactionoutput.RenderDetail(cmd.OutOrStdout(), account.AccountData(), transaction)
	return nil
}

func displayList(cmd *cobra.Command, account account.Account) error {
	transactions, err := account.Transactions(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed getting transactions: %w", err)
	}

	limit := lo.Must(cmd.Flags().GetInt("limit"))
	if lo.Must(cmd.Flags().GetBool("json")) {
		if err := transactionoutput.RenderJSON(cmd.OutOrStdout(), transactions, limit); err != nil {
			return err
		}
		return nil
	}

	transactionoutput.RenderTable(cmd.OutOrStdout(), transactions, limit)
	return nil
}

var transactionsCmd = &cobra.Command{
	Use:   "transactions [flags] [<transaction id>]",
	Short: "Lists your current account's transactions or display a single one if the id is present",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var transactionID string
		if len(args) > 0 {
			transactionID = args[0]
		}

		account, err := client.Account(cmd.Context(), viper.GetString("current-account"))
		if err != nil {
			return fmt.Errorf("failed fetching current account: %w", err)
		}

		if lo.Must(cmd.Flags().GetBool("sync")) {
			if _, err := account.LoadNewTransactions(cmd.Context()); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "failed fetching new transactions")
			}
		}

		if transactionID == "" {
			return displayList(cmd, account)
		}

		transactionIDNum, err := strconv.ParseInt(transactionID, 10, 64)
		if err != nil {
			return fmt.Errorf("failed parsing transaction ID: %w", err)
		}

		return displaySingle(cmd, transactionIDNum, account)
	},
}

func init() {
	transactionsCmd.Flags().Bool("sync", false, "Attempt sync before listing them")
	transactionsCmd.Flags().Int("limit", 20, "Limit the number of transactions to return")
	transactionsCmd.Flags().Bool("json", false, "Output results as JSON")

	rootCmd.AddCommand(transactionsCmd)
}
