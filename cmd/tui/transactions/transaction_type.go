package transactions

var transactionTypeTranslations = map[string]string{
	"Příjem převodem uvnitř banky":               "Incoming transfer within bank",
	"Platba převodem uvnitř banky":               "Outgoing transfer within bank",
	"Vklad pokladnou":                            "Cash deposit at counter",
	"Výběr pokladnou":                            "Cash withdrawal at counter",
	"Vklad v hotovosti":                          "Cash deposit",
	"Výběr v hotovosti":                          "Cash withdrawal",
	"Platba":                                     "Payment",
	"Příjem":                                     "Incoming payment",
	"Bezhotovostní platba":                       "Cashless payment",
	"Bezhotovostní příjem":                       "Cashless incoming payment",
	"Platba kartou":                              "Card payment",
	"Úrok z úvěru":                               "Loan interest",
	"Sankční poplatek":                           "Penalty fee",
	"Posel – předání":                            "Courier outgoing",
	"Posel – příjem":                             "Courier incoming",
	"Převod uvnitř konta":                        "Transfer within account",
	"Připsaný úrok":                              "Interest credited",
	"Vyplacený úrok":                             "Interest paid",
	"Odvod daně z úroků":                         "Interest tax",
	"Evidovaný úrok":                             "Recorded interest",
	"Poplatek":                                   "Fee",
	"Evidovaný poplatek":                         "Recorded fee",
	"Převod mezi bankovními konty (platba)":      "Interbank transfer outgoing",
	"Převod mezi bankovními konty (příjem)":      "Interbank transfer incoming",
	"Neidentifikovaná platba z bankovního konta": "Unidentified bank account payment",
	"Neidentifikovaný příjem na bankovní konto":  "Unidentified bank account incoming payment",
	"Vlastní platba z bankovního konta":          "Own bank account payment",
	"Vlastní příjem na bankovní konto":           "Own bank account incoming payment",
	"Vlastní platba pokladnou":                   "Own payment at counter",
	"Vlastní příjem pokladnou":                   "Own incoming payment at counter",
	"Opravný pohyb":                              "Correction",
	"Přijatý poplatek":                           "Fee received",
	"Platba v jiné měně":                         "Foreign currency payment",
	"Poplatek – platební karta":                  "Card fee",
	"Inkaso":                                     "Direct debit",
	"Inkaso ve prospěch účtu":                    "Direct debit incoming",
	"Inkaso z účtu":                              "Direct debit outgoing",
	"Příjem inkasa z cizí banky":                 "Direct debit received from another bank",
	"Okamžitá příchozí platba":                   "Instant incoming payment",
	"Okamžitá odchozí platba":                    "Instant outgoing payment",
	"Poplatek - pojištění hypotéky":              "Mortgage insurance fee",
	"Okamžitá příchozí Europlatba":               "Instant incoming euro payment",
	"Okamžitá odchozí Europlatba":                "Instant outgoing euro payment",
}

func translateTransactionType(transactionType string) string {
	if translation, ok := transactionTypeTranslations[transactionType]; ok {
		return translation
	}
	return transactionType
}
