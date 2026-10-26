package ledger

import "testing"

func TestLegsBalancePerCurrency(t *testing.T) {
	cash := account{1, "UZS"}
	humo := account{2, "UZS"}
	dollars := account{3, "USD"}
	tests := []struct {
		t        Type
		from     account
		to       *account
		amount   int64
		toAmount int64
	}{
		{Expense, cash, nil, 100, 0},
		{Income, cash, nil, 100, 0},
		{Transfer, cash, &humo, 100, 0},
		{Exchange, dollars, &cash, 100, 1_250_000},
		{Opening, cash, nil, 100, 0},
		{Opening, cash, nil, -100, 0}, // overdrawn card
	}
	for _, tt := range tests {
		sums := map[string]int64{}
		for _, l := range legsFor(tt.t, tt.amount, tt.from, tt.to, tt.toAmount) {
			sums[l.currency] += l.amount
			if l.amount == 0 {
				t.Errorf("%s: zero leg", tt.t)
			}
		}
		for cur, s := range sums {
			if s != 0 {
				t.Errorf("%s: %s legs sum to %d", tt.t, cur, s)
			}
		}
	}
}

func TestTransferNeverTouchesIncomeOrExpense(t *testing.T) {
	cash, humo, dollars := account{1, "UZS"}, account{2, "UZS"}, account{3, "USD"}
	for _, l := range append(legsFor(Transfer, 100, cash, &humo, 0), legsFor(Exchange, 100, dollars, &cash, 1_250_000)...) {
		if l.system == "income" || l.system == "expense" {
			t.Errorf("transfer/exchange leg hits %s", l.system)
		}
	}
}
