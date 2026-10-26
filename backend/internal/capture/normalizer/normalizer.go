// Package normalizer turns free text such as "taksi 25 ming" or "сорок пять тысяч вчера" into
// money amounts, dates and short descriptions. It is deterministic: no model ever decides an amount.
// Intent and category are left to later capture stages.
package normalizer

import (
	"math/big"
	"strings"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/money"
)

type Options struct {
	DefaultCurrency money.Currency // used when the text names none; UZS if empty
	Now             time.Time      // in the user's time zone; dates are relative to it
}

// Item is one amount found in the text with the words that describe it.
type Item struct {
	Description string
	Amount      money.Amount
	// CurrencyExplicit is false when DefaultCurrency was assumed.
	CurrencyExplicit bool
	// Date is midnight of the spending day in Options.Now's location.
	Date time.Time
	// Source is the amount as the user wrote it, for the confirmation card.
	Source string
	// NeedsReview marks amounts that are probably shorthand, e.g. "taksi 25" in UZS
	// most likely means 25 000. The caller should ask instead of auto-saving.
	NeedsReview bool
}

// Parse extracts every amount in text. One message may hold several expenses:
// "non 8 ming, sut 14 ming" yields two items.
func Parse(text string, opt Options) []Item {
	if opt.DefaultCurrency == "" {
		opt.DefaultCurrency = money.UZS
	}
	if opt.Now.IsZero() {
		opt.Now = time.Now()
	}
	toks := tokenize(text)
	cls := classify(toks)
	spans := findAmounts(cls)
	if len(spans) == 0 {
		return nil
	}

	days := 0
	for _, c := range cls {
		if c.kind == kDate {
			days = c.days
		}
	}
	y, m, d := opt.Now.AddDate(0, 0, days).Date()
	date := time.Date(y, m, d, 0, 0, 0, 0, opt.Now.Location())

	descs := describe(toks, cls, spans)
	items := make([]Item, 0, len(spans))
	for i, s := range spans {
		cur := s.currency
		if cur == "" {
			cur = opt.DefaultCurrency
		}
		minor, ok := toMinor(s.value, cur.MinorDigits())
		if !ok {
			continue
		}
		var src []string
		for j := s.from; j <= s.to; j++ {
			if k := cls[j].tok; j == s.from || k != cls[j-1].tok {
				src = append(src, toks[k].text)
			}
		}
		items = append(items, Item{
			Description:      descs[i],
			Amount:           money.Amount{Minor: minor, Currency: cur},
			CurrencyExplicit: s.currency != "",
			Date:             date,
			Source:           strings.Join(src, " "),
			NeedsReview:      !s.hasScale && cur == money.UZS && s.value.Cmp(big.NewRat(1000, 1)) < 0,
		})
	}
	return items
}

type ctok struct {
	entry
	tok int      // index into tokens
	val *big.Rat // kNum value
	sym bool     // written as a symbol, e.g. $
}

func classify(toks []token) []ctok {
	out := make([]ctok, 0, len(toks))
	for i, t := range toks {
		c := ctok{tok: i, sym: t.sym}
		switch {
		case t.sep:
			c.kind = kSep
		case t.num:
			var next entry
			if i+1 < len(toks) && !toks[i+1].num && !toks[i+1].sep {
				next, _ = lookup(toks[i+1].low)
			}
			if next.kind == kCounter {
				break
			}
			if v, ok := parseNumeral(t.low, next.kind == kScale); ok {
				c.kind, c.val = kNum, v
			}
		default:
			c.entry, _ = lookup(t.low)
		}
		out = append(out, c)
	}
	return out
}

type span struct {
	from, to  int // ctok indices, inclusive, including an attached currency
	value     *big.Rat
	currency  money.Currency
	hasScale  bool
	wordsOnly bool
}

// builder accumulates one number the way people say it: total holds completed scale groups
// ("ikki million"), cur the group being built ("besh yuz").
type builder struct {
	open          bool
	from, last    int
	curFrom       int
	total, cur    *big.Rat
	lastScale     int64
	hasScale      bool
	hasDigits     bool
	curWordsOnly  bool
	curHasHundred bool
	prev          kind
}

func findAmounts(cls []ctok) []span {
	var spans []span
	var b builder
	pendingCur, pendingAt := money.Currency(""), -2

	start := func(i int) {
		b = builder{open: true, from: i, last: i, curFrom: i, total: new(big.Rat), cur: new(big.Rat), curWordsOnly: true}
	}
	finish := func() {
		if !b.open {
			return
		}
		// "25 ming on taxi": a small spoken number after a scale is not part of the amount.
		if b.hasScale && b.cur.Sign() > 0 && b.curWordsOnly && !b.curHasHundred && b.cur.Cmp(big.NewRat(100, 1)) < 0 {
			b.cur.SetInt64(0)
			b.last = b.curFrom - 1
		}
		v := new(big.Rat).Add(b.total, b.cur)
		if v.Sign() > 0 {
			s := span{from: b.from, to: b.last, value: v, hasScale: b.hasScale, wordsOnly: !b.hasScale && !b.hasDigits}
			if pendingCur != "" && pendingAt == b.from-1 {
				s.from, s.currency = pendingAt, pendingCur
			}
			spans = append(spans, s)
		}
		b = builder{}
	}

	for i, c := range cls {
		switch c.kind {
		case kNum:
			if b.open && b.cur.Sign() != 0 {
				finish()
			}
			if !b.open {
				start(i)
			}
			b.cur.Set(c.val)
			b.curFrom, b.hasDigits, b.curWordsOnly, b.curHasHundred = i, true, false, false

		case kUnit:
			half := c.unit.Cmp(big.NewRat(1, 1)) < 0 || !c.unit.IsInt()
			if b.open && b.prev == kNum && !half {
				finish()
			}
			if !b.open {
				start(i)
			}
			if b.cur.Sign() == 0 {
				b.curFrom, b.curWordsOnly, b.curHasHundred = i, true, false
			}
			b.cur.Add(b.cur, c.unit)

		case kHundred:
			if !b.open {
				start(i)
			}
			if b.cur.Sign() == 0 {
				b.cur.SetInt64(1)
				b.curFrom, b.curWordsOnly = i, true
			}
			b.cur.Mul(b.cur, big.NewRat(100, 1))
			b.curHasHundred = true

		case kScale:
			if c.needsNum && !(b.open && b.prev == kNum) {
				finish()
				continue
			}
			if !b.open {
				start(i)
			} else if b.lastScale != 0 && c.scale >= b.lastScale {
				// "8 ming 14 ming" is two amounts: split before the second number.
				cur, cf := new(big.Rat).Set(b.cur), b.curFrom
				if cur.Sign() == 0 {
					finish()
					start(i)
				} else {
					b.cur.SetInt64(0)
					b.last = cf - 1
					finish()
					start(cf)
					b.cur = cur
					for j := cf; j < i; j++ {
						b.hasDigits = b.hasDigits || cls[j].kind == kNum
					}
				}
			}
			if c.unit != nil { // полмиллиона
				b.cur.Add(b.cur, c.unit)
			}
			m := new(big.Rat).Set(b.cur)
			if m.Sign() == 0 {
				m.SetInt64(1)
			}
			b.total.Add(b.total, m.Mul(m, big.NewRat(c.scale, 1)))
			b.cur.SetInt64(0)
			b.lastScale, b.hasScale = c.scale, true

		case kCurrency:
			finish()
			n := len(spans)
			prevAdjacent := n > 0 && spans[n-1].to == i-1 && spans[n-1].currency == ""
			nextIsNum := i+1 < len(cls) && (cls[i+1].kind == kNum || cls[i+1].kind == kUnit)
			// "$20" and "usd 20" put the currency first; "20$" and "20 so'm" after.
			if prevAdjacent && !(c.sym && nextIsNum) {
				spans[n-1].to, spans[n-1].currency = i, c.currency
			} else if nextIsNum {
				pendingCur, pendingAt = c.currency, i
			}
			continue

		default:
			finish()
			continue
		}
		b.last, b.prev = i, c.kind
	}
	finish()

	// Word-only numbers need a scale or a currency: "bir kofe" is one coffee,
	// "bir ming" and "yigirma dollar" are money.
	kept := spans[:0]
	for _, s := range spans {
		if !s.wordsOnly || s.currency != "" {
			kept = append(kept, s)
		}
	}
	return kept
}

// describe returns one description per span. Words before the first amount mean the user writes
// "taksi 25 ming, non 8 ming"; otherwise "25 ming taksiga, 8 ming non" and words follow amounts.
func describe(toks []token, cls []ctok, spans []span) []string {
	skip := make([]bool, len(toks))
	for _, c := range cls {
		if c.kind == kCurrency || c.kind == kDate || c.kind == kSep {
			skip[c.tok] = true
		}
	}
	for i, t := range toks {
		if t.sym || filler[t.low] {
			skip[i] = true
		}
	}
	words := func(from, to int) []string { // token range [from, to)
		var w []string
		for i := max(from, 0); i < to && i < len(toks); i++ {
			if !skip[i] {
				w = append(w, toks[i].text)
			}
		}
		return w
	}
	tokFrom := func(s span) int { return cls[s.from].tok }
	tokTo := func(s span) int { return cls[s.to].tok + 1 }

	out := make([]string, len(spans))
	prefixStyle := len(words(0, tokFrom(spans[0]))) > 0
	for i, s := range spans {
		var w []string
		if prefixStyle {
			lo := 0
			if i > 0 {
				lo = tokTo(spans[i-1])
			}
			w = words(lo, tokFrom(s))
			if i == len(spans)-1 { // trailing words belong to the last item: "taksi 25 ming uyga"
				w = append(w, words(tokTo(s), len(toks))...)
			}
		} else {
			hi := len(toks)
			if i+1 < len(spans) {
				hi = tokFrom(spans[i+1])
			}
			w = words(tokTo(s), hi)
		}
		out[i] = strings.Join(w, " ")
	}
	return out
}

func toMinor(r *big.Rat, digits int) (int64, bool) {
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	x := new(big.Rat).Mul(r, new(big.Rat).SetInt(scale))
	q, rem := new(big.Int).QuoRem(x.Num(), x.Denom(), new(big.Int))
	if rem.Mul(rem, big.NewInt(2)).Cmp(x.Denom()) >= 0 { // half up
		q.Add(q, big.NewInt(1))
	}
	if !q.IsInt64() {
		return 0, false
	}
	return q.Int64(), true
}
