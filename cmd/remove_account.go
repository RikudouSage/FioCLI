package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.chrastecky.dev/fio/fiocli/cmd/helper"
)

var removeAccountCmd = &cobra.Command{
	Use:   "remove-account [flags] [<account number>]",
	Short: "Removes an account from your local database",
	Aliases: []string{
		"remove",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accountNumber, err := helper.GetAccountFromArgsOrInteractively(args, cmd, client)
		if err != nil {
			return err
		}
		if accountNumber == "" {
			return fmt.Errorf("account number is empty")
		}

		if err := client.RemoveAccount(cmd.Context(), accountNumber); err != nil {
			return fmt.Errorf("failed removing account: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Account removed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeAccountCmd)
}
