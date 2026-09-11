package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.chrastecky.dev/fio/fiocli/cmd/helper"
)

var switchAccountCmd = &cobra.Command{
	Use:   "switch-account [flags] [<account number>]",
	Short: "Switch the currently selected account",
	Long:  "Switch the currently selected account, the currently selected account is the one all account operations are being done on",
	Aliases: []string{
		"switch",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		newAccount, err := helper.GetAccountFromArgsOrInteractively(args, cmd, client)
		if err != nil {
			return err
		}

		if newAccount == "" {
			return fmt.Errorf("account number is empty")
		}

		_, err = client.Account(cmd.Context(), newAccount)
		if err != nil {
			return fmt.Errorf("failed getting account: %w", err)
		}

		if err := helper.SetCurrentAccount(cfgFile, newAccount); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Switched account successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(switchAccountCmd)
}
