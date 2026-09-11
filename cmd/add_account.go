package cmd

import (
	"errors"
	"fmt"

	"charm.land/huh/v2"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var addAccountCmd = &cobra.Command{
	Use:   "add-account [flags]",
	Short: "Adds a new account to your local database",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := lo.Must(cmd.Flags().GetString("api-key"))
		if apiKey != "" {
			fmt.Fprintln(cmd.OutOrStdout(), "Warning: command line argument API key provided, please prefer asking for it interactively.")
		} else {
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("API key").
						EchoMode(huh.EchoModePassword).
						Value(&apiKey),
				),
			).
				WithInput(cmd.InOrStdin()).
				WithOutput(cmd.OutOrStdout())

			if err := form.Run(); err != nil {
				return fmt.Errorf("failed reading api key: %w", err)
			}
		}

		if apiKey == "" {
			return errors.New("no api key provided")
		}

		longPull := lo.Must(cmd.Flags().GetBool("long-pull"))
		account, err := client.RegisterAccount(cmd.Context(), apiKey, longPull)
		if err != nil {
			return fmt.Errorf("failed adding an account, is the api key correct? %w", err)
		}

		tbl := table.New("Account number", "Bank code", "Currency", "IBAN", "BIC")
		tbl.WithHeaderFormatter(color.New(color.FgGreen, color.Underline).SprintfFunc())
		tbl.WithWriter(cmd.OutOrStdout())
		tbl.AddRow(account.AccountNumber, account.BankCode, account.Currency, account.IBAN, account.BIC)
		tbl.Print()

		fmt.Fprintln(cmd.OutOrStdout(), "account successfully added")

		return nil
	},
}

func init() {
	addAccountCmd.Flags().String("api-key", "", "the API key to use for the new client, prefer not specifying this option and you'll be asked interactively instead")
	addAccountCmd.Flags().Bool("long-pull", false, "when present, the initial sync will try to synchronize the last 10 years of data instead of last 90 days, note that you have to unlock this option for your api key in your internet banking and the permission is only valid for 10 minutes")

	viper.BindPFlag("api-key", addAccountCmd.Flags().Lookup("api-key"))

	rootCmd.AddCommand(addAccountCmd)
}
