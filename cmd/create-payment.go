package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.chrastecky.dev/fio-api/fio/dto"
	fiotypes "go.chrastecky.dev/fio-api/fio/types"
)

var createPaymentCmd = &cobra.Command{
	Use:   "create-payment [domestic] <account>/<bank-code> <amount> [flags]",
	Short: "Create an outgoing payment (defaults to a domestic payment)",
	Args:  cobra.RangeArgs(2, 3),
	Aliases: []string{
		"create",
		"pay",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		payment, err := domesticPaymentFromCommand(cmd, args)
		if err != nil {
			return err
		}

		account, err := client.Account(cmd.Context(), viper.GetString("current-account"))
		if err != nil {
			return fmt.Errorf("failed fetching current account: %w", err)
		}
		if _, err := account.CreateDomesticPayment(cmd.Context(), payment); err != nil {
			return fmt.Errorf("failed creating payment: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Payment successfully created, please confirm it your usual way.")
		return nil
	},
}

func domesticPaymentFromCommand(cmd *cobra.Command, args []string) (dto.DomesticTransaction, error) {
	targetIndex := 0
	if len(args) == 3 {
		if !strings.EqualFold(args[0], "domestic") {
			return dto.DomesticTransaction{}, fmt.Errorf("unsupported payment type %q; only 'domestic' is supported", args[0])
		}
		targetIndex = 1
	}

	accountTo, bankCode, found := strings.Cut(args[targetIndex], "/")
	accountTo, bankCode = strings.TrimSpace(accountTo), strings.TrimSpace(bankCode)
	if !found || accountTo == "" || bankCode == "" || strings.Contains(bankCode, "/") {
		return dto.DomesticTransaction{}, errors.New("target must have the form <account>/<bank-code>")
	}

	amountText := args[targetIndex+1]
	amount, err := decimal.NewFromString(amountText)
	if err != nil {
		return dto.DomesticTransaction{}, fmt.Errorf("invalid amount %q: %w", amountText, err)
	}
	if !amount.IsPositive() {
		return dto.DomesticTransaction{}, errors.New("amount must be greater than zero")
	}

	paymentType, err := parseDomesticPaymentType(stringFlag(cmd, "payment-type"))
	if err != nil {
		return dto.DomesticTransaction{}, err
	}
	payment := dto.DomesticTransaction{
		AccountTo:   accountTo,
		BankCode:    bankCode,
		Amount:      amount,
		Currency:    strings.ToUpper(stringFlag(cmd, "currency")),
		PaymentType: paymentType,
	}

	date := stringFlag(cmd, "date")
	if date == "" {
		payment.Date = fiotypes.Date(time.Now())
	} else if err := payment.Date.UnmarshalText([]byte(date)); err != nil {
		return dto.DomesticTransaction{}, fmt.Errorf("invalid payment date %q: %w", date, err)
	}

	payment.ConstantSymbol = optionalStringFlag(cmd, "constant-symbol")
	payment.VariableSymbol = optionalStringFlag(cmd, "variable-symbol")
	payment.SpecificSymbol = optionalStringFlag(cmd, "specific-symbol")
	payment.MessageForRecipient = optionalStringFlag(cmd, "message")
	payment.Comment = optionalStringFlag(cmd, "comment")
	return payment, nil
}

func parseDomesticPaymentType(value string) (dto.DomesticPaymentType, error) {
	switch strings.ToLower(value) {
	case "standard":
		return dto.DomesticPaymentStandard, nil
	case "priority":
		return dto.DomesticPaymentPriority, nil
	case "direct-debit":
		return dto.DomesticPaymentDirectDebit, nil
	default:
		return "", fmt.Errorf("invalid payment type %q; expected 'standard', 'priority', or 'direct-debit'", value)
	}
}

func stringFlag(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return value
}

func optionalStringFlag(cmd *cobra.Command, name string) *string {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	value := stringFlag(cmd, name)
	return &value
}

func init() {
	createPaymentCmd.Flags().String("currency", "", "ISO 4217 currency code (defaults to the current account's currency)")
	createPaymentCmd.Flags().String("constant-symbol", "", "constant symbol")
	createPaymentCmd.Flags().String("variable-symbol", "", "variable symbol")
	createPaymentCmd.Flags().String("specific-symbol", "", "specific symbol")
	createPaymentCmd.Flags().String("date", "", "requested payment date in YYYY-MM-DD format (defaults to today)")
	createPaymentCmd.Flags().String("message", "", "message for the recipient")
	createPaymentCmd.Flags().String("comment", "", "payer-side comment")
	createPaymentCmd.Flags().String("payment-type", "standard", "processing type: standard, priority, or direct-debit")

	rootCmd.AddCommand(createPaymentCmd)
}
