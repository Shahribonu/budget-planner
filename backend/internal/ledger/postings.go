package ledger

// leg is one side of a transaction before it is written: either a user's asset account
// or a system account looked up by key and currency.
type leg struct {
	asset    int64  // account id (asset, receivable or payable); 0 for system accounts
	system   string // income, expense, opening, fx
	amount   int64  // debit positive, credit negative
	currency string
}

var systemKind = map[string]string{
	"income":  "income",
	"expense": "expense",
	"opening": "equity",
	"fx":      "equity",
}

// legsFor turns a validated transaction into balanced postings.
// Transfers and exchanges touch only asset and equity accounts, so they never
// show up as income or expense.
func legsFor(t Type, amount int64, from account, to *account, toAmount int64) []leg {
	switch t {
	case Expense:
		return []leg{
			{asset: from.id, amount: -amount, currency: from.currency},
			{system: "expense", amount: amount, currency: from.currency},
		}
	case Income:
		return []leg{
			{asset: from.id, amount: amount, currency: from.currency},
			{system: "income", amount: -amount, currency: from.currency},
		}
	case Transfer:
		return []leg{
			{asset: from.id, amount: -amount, currency: from.currency},
			{asset: to.id, amount: amount, currency: to.currency},
		}
	case Exchange:
		return []leg{
			{asset: from.id, amount: -amount, currency: from.currency},
			{system: "fx", amount: amount, currency: from.currency},
			{system: "fx", amount: -toAmount, currency: to.currency},
			{asset: to.id, amount: toAmount, currency: to.currency},
		}
	case Opening:
		return []leg{
			{asset: from.id, amount: amount, currency: from.currency},
			{system: "opening", amount: -amount, currency: from.currency},
		}
	}
	return nil
}

type account struct {
	id       int64
	currency string
}
