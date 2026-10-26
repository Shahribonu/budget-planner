// Package money holds integer amounts with an explicit currency.
// Amounts are always stored in ISO 4217 minor units and never mixed across currencies.
package money

import (
	"strconv"
	"strings"
)

type Currency string

const (
	UZS Currency = "UZS"
	USD Currency = "USD"
)

var minorDigits = map[Currency]int{
	UZS: 2,
	USD: 2,
}

// MinorDigits returns the ISO 4217 exponent of c (2 when unknown).
func (c Currency) MinorDigits() int {
	if d, ok := minorDigits[c]; ok {
		return d
	}
	return 2
}

// Amount is a value in minor units of Currency.
type Amount struct {
	Minor    int64
	Currency Currency
}

// New builds an Amount from whole major units, e.g. New(45000, UZS).
func New(major int64, c Currency) Amount {
	m := major
	for range c.MinorDigits() {
		m *= 10
	}
	return Amount{Minor: m, Currency: c}
}

// String formats the amount for logs and tests, e.g. "45 000 UZS" or "12.50 USD".
// User-facing formatting lives in i18n.
func (a Amount) String() string {
	d := a.Currency.MinorDigits()
	neg := a.Minor < 0
	s := strconv.FormatInt(a.Minor, 10)
	if neg {
		s = s[1:]
	}
	for len(s) <= d {
		s = "0" + s
	}
	major, frac := s[:len(s)-d], s[len(s)-d:]

	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, r := range major {
		if i > 0 && (len(major)-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	if strings.Trim(frac, "0") != "" {
		b.WriteByte('.')
		b.WriteString(frac)
	}
	b.WriteByte(' ')
	b.WriteString(string(a.Currency))
	return b.String()
}
