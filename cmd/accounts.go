package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.chrastecky.dev/fio/fiocli/cmd/helper"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts [flags]",
	Short: "Lists your locally configured accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		accounts, err := client.Accounts(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed listing accounts: %w", err)
		}

		if len(accounts) == 0 {
			fmt.Println("No accounts found.")
			return nil
		}

		showAPIKeys := lo.Must(cmd.Flags().GetBool("show-api-keys"))
		headers := []any{"Account number", "Bank code", "Currency", "IBAN", "BIC", "Currently selected"}
		if showAPIKeys {
			headers = append(headers, "API Key")
		}

		headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
		tbl := table.New(headers...)
		tbl.WithHeaderFormatter(headerFmt)
		tbl.WithWriter(cmd.OutOrStdout())

		for _, account := range accounts {
			values := []any{
				account.AccountNumber,
				account.BankCode,
				account.Currency,
				account.IBAN,
				account.BIC,
				helper.FormatBool(account.AccountNumber == viper.GetString("current-account")),
			}
			if showAPIKeys {
				values = append(values, account.APIKey)
			}
			tbl.AddRow(values...)
		}

		tbl.Print()
		return nil
	},
}

func init() {
	accountsCmd.Flags().Bool("show-api-keys", false, "show API keys when listing secrets")

	rootCmd.AddCommand(accountsCmd)
}
