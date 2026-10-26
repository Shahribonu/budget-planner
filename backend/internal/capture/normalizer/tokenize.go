package normalizer

import (
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"
)

type token struct {
	text        string // as written, for descriptions
	low         string // lowercased, apostrophes unified
	num         bool
	sym         bool // $ and other single-rune symbols
	sep         bool
	spaceBefore bool
	grouped     bool // digit groups already joined: "2 500" → "2500"
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

func isApostrophe(r rune) bool {
	switch r {
	case '\'', '‘', '’', 'ʻ', 'ʼ', '`', '´':
		return true
	}
	return false
}

func tokenize(s string) []token {
	var toks []token
	space := false
	emit := func(t token) {
		t.spaceBefore = space
		toks = append(toks, t)
		space = false
	}
	for i := 0; i < len(s); {
		r, w := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '\n':
			emit(token{text: "\n", sep: true})
			i += w
		case unicode.IsSpace(r):
			space = true
			i += w
		case isDigit(r):
			j := i + w
			for j < len(s) {
				r2, w2 := utf8.DecodeRuneInString(s[j:])
				if isDigit(r2) || ((r2 == '.' || r2 == ',') && j+w2 < len(s) && isDigit(rune(s[j+w2]))) {
					j += w2
					continue
				}
				break
			}
			emit(token{text: s[i:j], low: s[i:j], num: true})
			i = j
		case unicode.IsLetter(r):
			j := i + w
			for j < len(s) {
				r2, w2 := utf8.DecodeRuneInString(s[j:])
				if unicode.IsLetter(r2) || unicode.IsMark(r2) {
					j += w2
					continue
				}
				if isApostrophe(r2) && j+w2 < len(s) {
					if r3, _ := utf8.DecodeRuneInString(s[j+w2:]); unicode.IsLetter(r3) {
						j += w2
						continue
					}
				}
				break
			}
			emit(token{text: s[i:j], low: normWord(s[i:j])})
			i = j
		case r == '$':
			emit(token{text: "$", low: "$", sym: true})
			i += w
		default:
			emit(token{text: s[i : i+w], sep: true})
			i += w
		}
	}
	return groupDigits(toks)
}

func normWord(w string) string {
	return strings.Map(func(r rune) rune {
		if isApostrophe(r) {
			return '\''
		}
		return unicode.ToLower(r)
	}, w)
}

// groupDigits joins space-separated thousands groups: "2 500 000" → "2500000".
func groupDigits(toks []token) []token {
	var out []token
	for _, t := range toks {
		if n := len(out); n > 0 && t.num && t.spaceBefore && len(t.low) == 3 && allDigits(t.low) {
			prev := &out[n-1]
			if prev.num && (prev.grouped || (len(prev.low) <= 3 && allDigits(prev.low))) {
				prev.text += " " + t.text
				prev.low += t.low
				prev.grouped = true
				continue
			}
		}
		out = append(out, t)
	}
	return out
}

func allDigits(s string) bool {
	for _, r := range s {
		if !isDigit(r) {
			return false
		}
	}
	return s != ""
}

// parseNumeral reads a digit literal. A single "." or "," followed by exactly three digits is a
// thousands separator (45.000) unless a scale word follows (1.500 mln); otherwise it is a decimal
// point (0.5, 45,5). Repeated identical separators are thousands separators (1.500.000).
func parseNumeral(s string, beforeScale bool) (*big.Rat, bool) {
	if len(s) > 18 {
		return nil, false
	}
	dots, commas := strings.Count(s, "."), strings.Count(s, ",")
	switch {
	case dots+commas == 1:
		last := strings.IndexAny(s, ".,")
		if len(s)-last-1 == 3 && !beforeScale {
			s = s[:last] + s[last+1:]
		} else {
			s = s[:last] + "." + s[last+1:]
		}
	case dots == 0 || commas == 0:
		s = strings.NewReplacer(".", "", ",", "").Replace(s)
	default:
		last := strings.LastIndexAny(s, ".,")
		s = strings.NewReplacer(".", "", ",", "").Replace(s[:last]) + "." + s[last+1:]
	}
	r, ok := new(big.Rat).SetString(s)
	return r, ok
}
