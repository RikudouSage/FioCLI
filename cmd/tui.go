package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.chrastecky.dev/fio/fiocli/cmd/tui"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Open the interactive account transaction browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		account, err := client.Account(cmd.Context(), viper.GetString("current-account"))
		if err != nil {
			return fmt.Errorf("failed fetching current account: %w", err)
		}

		transactions, err := account.Transactions(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed getting transactions: %w", err)
		}

		if err := tui.Run(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), account.AccountData(), transactions); err != nil {
			return fmt.Errorf("failed rendering transactions: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
