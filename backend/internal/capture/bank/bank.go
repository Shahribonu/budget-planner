// Package bank parses card notifications that users forward from bank bots or SMS.
// Parsers are deterministic; a message no parser recognizes falls back to normal chat
// logging. Card numbers are reduced to their last four digits.
package bank

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/money"
)

type Direction string

const (
	Debit  Direction = "debit"  // money left the card
	Credit Direction = "credit" // money arrived
)

type Kind string

const (
	Payment     Kind = "payment" // purchase or bill
	TopUp       Kind = "topup"   // money in: transfer from someone, own account or salary
	TransferIn  Kind = "transfer_in"
	TransferOut Kind = "transfer_out"
	Withdrawal  Kind = "withdrawal" // cash from an ATM
	Refund      Kind = "refund"
	Fee         Kind = "fee"
	Unknown     Kind = "unknown"
)

type Notification struct {
	Bank      string // "humo"
	Kind      Kind
	Direction Direction
	Amount    money.Amount // always positive
	Merchant  string       // as written, e.g. "AAB CLICK QR OPLATA"
	CardMask  string       // last four digits
	At        time.Time    // bank time, Tashkent
	Balance   *money.Amount
}

var tashkent = time.FixedZone("UZT", 5*3600)

var (
	humoCard   = regexp.MustCompile(`(?i)HUMO\s*CARD\s*\*+\s*(\d{4})`)
	signAmount = regexp.MustCompile(`([➕➖+\-−—])\s*([\d .\x{00a0}\x{202f}]+,\d{2})\s*(UZS|USD|сум|so'm)`)
	balance    = regexp.MustCompile(`💰\s*([\d .\x{00a0}\x{202f}]+,\d{2})\s*(UZS|USD)`)
	place      = regexp.MustCompile(`📍\s*(.+)`)
	stamp      = regexp.MustCompile(`(\d{1,2}):(\d{2})\s+(\d{2})\.(\d{2})\.(\d{4})`)
)

// kinds maps the title line of a notification to a kind, in Uzbek and Russian.
var kinds = []struct {
	words []string
	kind  Kind
}{
	{[]string{"to'ldirish", "пополнение", "to'ldirildi"}, TopUp},
	{[]string{"to'lov", "оплата", "xarid", "покупка"}, Payment},
	{[]string{"qaytarish", "возврат"}, Refund},
	{[]string{"naqd", "yechib", "снятие", "наличн", "bankomat", "atm"}, Withdrawal},
	{[]string{"komissiya", "комиссия"}, Fee},
	{[]string{"o'tkazma", "перевод"}, TransferOut},
}

// Parse recognizes a notification. ok is false when the text is not one.
func Parse(text string) (n Notification, ok bool) {
	text = strings.NewReplacer("‘", "'", "’", "'", "ʻ", "'", "ʼ", "'", "`", "'").Replace(text)
	card := humoCard.FindStringSubmatch(text)
	amt := signAmount.FindStringSubmatch(text)
	if card == nil || amt == nil {
		return n, false
	}
	minor, ok := parseAmount(amt[2])
	if !ok || minor <= 0 {
		return n, false
	}
	n = Notification{Bank: "humo", CardMask: card[1], Amount: money.Amount{Minor: minor, Currency: currency(amt[3])}}
	n.Direction = Debit
	if amt[1] == "➕" || amt[1] == "+" {
		n.Direction = Credit
	}

	if m := place.FindStringSubmatch(text); m != nil {
		n.Merchant = strings.TrimSpace(m[1])
	}
	if m := balance.FindStringSubmatch(text); m != nil {
		if b, ok := parseAmount(m[1]); ok {
			n.Balance = &money.Amount{Minor: b, Currency: currency(m[2])}
		}
	}
	if m := stamp.FindStringSubmatch(text); m != nil {
		h, _ := strconv.Atoi(m[1])
		mi, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		mo, _ := strconv.Atoi(m[4])
		y, _ := strconv.Atoi(m[5])
		n.At = time.Date(y, time.Month(mo), d, h, mi, 0, 0, tashkent)
	}

	n.Kind = Unknown
	title := strings.ToLower(firstLine(text))
	for _, k := range kinds {
		for _, w := range k.words {
			if strings.Contains(title, w) {
				n.Kind = k.kind
				break
			}
		}
		if n.Kind != Unknown {
			break
		}
	}
	// The sign is authoritative: an "o'tkazma" that adds money is a transfer in.
	switch {
	case n.Kind == TransferOut && n.Direction == Credit:
		n.Kind = TransferIn
	case n.Kind == Unknown && n.Direction == Credit:
		n.Kind = TopUp
	case n.Kind == Unknown:
		n.Kind = Payment
	}
	return n, true
}

func firstLine(text string) string {
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(l)
		if l != "" && !strings.EqualFold(l, "HUMO Card") {
			return l
		}
	}
	return ""
}

func currency(s string) money.Currency {
	if strings.EqualFold(s, "USD") {
		return money.USD
	}
	return money.UZS
}

// parseAmount reads "37.470,00": dots or spaces group thousands, a comma marks tiyin.
func parseAmount(s string) (int64, bool) {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '.', ' ', ' ', ' ':
			return -1
		}
		return r
	}, s)
	whole, frac, ok := strings.Cut(s, ",")
	if !ok || len(frac) != 2 {
		return 0, false
	}
	w, err1 := strconv.ParseInt(whole, 10, 64)
	f, err2 := strconv.ParseInt(frac, 10, 64)
	if err1 != nil || err2 != nil || w > 1e13 {
		return 0, false
	}
	return w*100 + f, true
}

// MerchantName cleans a merchant line for descriptions and category learning:
// "IP OOO JETI ASPAN>TA" → "Jeti Aspan", "AAB CLICK QR OPLATA" → "Click QR".
func MerchantName(raw string) string {
	s := raw
	if i := strings.Index(s, ">"); i >= 0 {
		s = s[:i] // ">Toshkent" is the city, often cut off
	}
	words := strings.Fields(strings.ToUpper(s))
	drop := map[string]bool{"IP": true, "OOO": true, "MCHJ": true, "XK": true, "YATT": true, "AJ": true, "AAB": true, "OPLATA": true, "TOLOV": true, "UZ": true, "LLC": true}
	var out []string
	for _, w := range words {
		if !drop[w] {
			out = append(out, titleCase(w))
		}
	}
	if len(out) == 0 {
		return strings.TrimSpace(raw)
	}
	return strings.Join(out, " ")
}

func titleCase(w string) string {
	switch w {
	case "QR", "ATM", "UZCARD", "HUMO":
		return w
	}
	r := []rune(strings.ToLower(w))
	if len(r) > 0 {
		r[0] = []rune(strings.ToUpper(string(r[0])))[0]
	}
	return string(r)
}
