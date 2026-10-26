package money

import "testing"

func TestString(t *testing.T) {
	tests := []struct {
		in   Amount
		want string
	}{
		{New(45000, UZS), "45 000 UZS"},
		{New(12000000, UZS), "12 000 000 UZS"},
		{New(0, UZS), "0 UZS"},
		{Amount{Minor: 1250, Currency: USD}, "12.50 USD"},
		{Amount{Minor: 5, Currency: USD}, "0.05 USD"},
		{Amount{Minor: -10000000, Currency: UZS}, "-100 000 UZS"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("%+v.String() = %q, want %q", tt.in, got, tt.want)
		}
	}
}
