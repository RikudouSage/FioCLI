package transactions

import (
	"encoding/json"
	"fmt"
	"io"

	"go.chrastecky.dev/fio-client/fioclient/model"
)

// RenderJSON writes transactions as an indented JSON array.
func RenderJSON(output io.Writer, txs []model.Transaction, limit int) error {
	if txs == nil {
		txs = []model.Transaction{}
	}
	if limit > 0 && len(txs) > limit {
		txs = txs[:limit]
	}

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(txs); err != nil {
		return fmt.Errorf("failed encoding transactions as JSON: %w", err)
	}
	return nil
}

// RenderTransactionJSON writes one transaction as indented JSON.
func RenderTransactionJSON(output io.Writer, transaction model.Transaction) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(transaction); err != nil {
		return fmt.Errorf("failed encoding transaction as JSON: %w", err)
	}
	return nil
}
