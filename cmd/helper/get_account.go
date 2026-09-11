package helper

import (
	"fmt"

	"charm.land/huh/v2"
	"github.com/fatih/color"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.chrastecky.dev/fio-client/fioclient"
	"go.chrastecky.dev/fio-client/fioclient/model"
)

func GetAccountFromArgsOrInteractively(args []string, cmd *cobra.Command, client fioclient.Client) (string, error) {
	var result string
	if len(args) > 0 {
		result = args[0]
	} else {
		allAccounts, err := client.Accounts(cmd.Context())
		if err != nil {
			allAccounts = make([]model.Account, 0)
		}

		options := lo.Map(allAccounts, func(item model.Account, _ int) huh.Option[string] {
			name := item.AccountNumber
			if viper.GetString("current-account") == name {
				gray := color.New(color.Faint).SprintFunc()
				name += gray(" (current)")
			}

			return huh.NewOption(name, item.AccountNumber)
		})
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Account Number").
					Options(options...).
					Value(&result),
			),
		).
			WithInput(cmd.InOrStdin()).
			WithOutput(cmd.OutOrStdout())

		if err := form.Run(); err != nil {
			return "", fmt.Errorf("failed reading account number: %w", err)
		}
	}

	return result, nil
}
