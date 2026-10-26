package normalizer

import (
	"math/big"
	"strings"

	"github.com/Shahribonu/budget-planner/backend/internal/money"
)

type kind int

const (
	kOther    kind = iota
	kNum           // digit literal: 45, 45.000, 0.5
	kUnit          // additive number word: besh, qirq, сорок, двести, yarim, полтора
	kHundred       // multiplier for the preceding unit: yuz, юз, hundred
	kScale         // ming, тысяч, million, k
	kCurrency      // so'm, $, бакс
	kDate          // kecha, вчера
	kCounter       // "2 ta non": the number before it is a quantity, not money
	kSep           // punctuation that separates items
)

type entry struct {
	kind     kind
	unit     *big.Rat
	scale    int64
	needsNum bool // scale only valid right after a digit literal: 45k, 45к, 45т
	currency money.Currency
	days     int
}

var vocab = map[string]entry{}

func init() {
	units := map[int64][]string{
		1:   {"bir", "бир", "один", "одна", "одну", "one"},
		2:   {"ikki", "икки", "два", "две", "two"},
		3:   {"uch", "уч", "три", "three"},
		4:   {"to'rt", "тўрт", "торт", "четыре", "four"},
		5:   {"besh", "беш", "пять", "five"},
		6:   {"olti", "олти", "шесть", "six"},
		7:   {"yetti", "етти", "семь", "seven"},
		8:   {"sakkiz", "саккиз", "восемь", "eight"},
		9:   {"to'qqiz", "тўққиз", "тоққиз", "девять", "nine"},
		10:  {"o'n", "ўн", "он", "десять", "ten"},
		11:  {"одиннадцать", "eleven"},
		12:  {"двенадцать", "twelve"},
		13:  {"тринадцать", "thirteen"},
		14:  {"четырнадцать", "fourteen"},
		15:  {"пятнадцать", "fifteen"},
		16:  {"шестнадцать", "sixteen"},
		17:  {"семнадцать", "seventeen"},
		18:  {"восемнадцать", "eighteen"},
		19:  {"девятнадцать", "nineteen"},
		20:  {"yigirma", "йигирма", "двадцать", "twenty"},
		30:  {"o'ttiz", "ўттиз", "оттиз", "тридцать", "thirty"},
		40:  {"qirq", "қирқ", "кирк", "сорок", "forty"},
		50:  {"ellik", "эллик", "пятьдесят", "fifty"},
		60:  {"oltmish", "олтмиш", "шестьдесят", "sixty"},
		70:  {"yetmish", "етмиш", "семьдесят", "seventy"},
		80:  {"sakson", "саксон", "восемьдесят", "eighty"},
		90:  {"to'qson", "тўқсон", "тоқсон", "девяносто", "ninety"},
		100: {"сто"},
		200: {"двести"},
		300: {"триста"},
		400: {"четыреста"},
		500: {"пятьсот"},
		600: {"шестьсот"},
		700: {"семьсот"},
		800: {"восемьсот"},
		900: {"девятьсот"},
	}
	for v, words := range units {
		for _, w := range words {
			add(w, entry{kind: kUnit, unit: big.NewRat(v, 1)})
		}
	}
	for _, w := range []string{"yarim", "ярим", "пол", "half"} {
		add(w, entry{kind: kUnit, unit: big.NewRat(1, 2)})
	}
	for _, w := range []string{"полтора", "полторы"} {
		add(w, entry{kind: kUnit, unit: big.NewRat(3, 2)})
	}
	for _, w := range []string{"yuz", "юз", "hundred"} {
		add(w, entry{kind: kHundred})
	}

	scales := map[int64][]string{
		1e3: {"ming", "минг", "тысяча", "тысячи", "тысяч", "тысячу", "тыща", "тыщи", "тыщ", "тыс", "thousand"},
		1e6: {"million", "millon", "milyon", "mln", "миллион", "миллиона", "миллионов", "млн", "лям", "ляма", "лямов"},
		1e9: {"milliard", "mlrd", "миллиард", "миллиарда", "миллиардов", "млрд", "billion"},
	}
	for v, words := range scales {
		for _, w := range words {
			add(w, entry{kind: kScale, scale: v})
		}
	}
	for _, w := range []string{"k", "к", "т"} {
		add(w, entry{kind: kScale, scale: 1e3, needsNum: true})
	}

	currencies := map[money.Currency][]string{
		money.UZS: {"so'm", "сўм", "сум", "сума", "сумов", "сом", "sum", "uzs"},
		money.USD: {"$", "usd", "dollar", "dollars", "dollor", "доллар", "доллара", "долларов", "долл", "бакс", "бакса", "баксов", "баксы"},
	}
	for c, words := range currencies {
		for _, w := range words {
			add(w, entry{kind: kCurrency, currency: c})
		}
	}

	for _, w := range []string{"bugun", "bugungi", "бугун", "бугунги", "сегодня", "today"} {
		add(w, entry{kind: kDate, days: 0})
	}
	for _, w := range []string{"kecha", "kechagi", "кеча", "кечаги", "вчера", "yesterday"} {
		add(w, entry{kind: kDate, days: -1})
	}
	add("позавчера", entry{kind: kDate, days: -2})

	for _, w := range []string{"ta", "та", "dona", "дона", "шт", "штук", "штуки", "штука", "x", "pcs", "kg", "кг", "kilo", "кило", "litr", "литр", "литра"} {
		add(w, entry{kind: kCounter})
	}

	for _, w := range []string{
		"uchun", "учун", "va", "ва", "ham", "ҳам",
		"oldim", "олдим", "to'ladim", "тўладим", "sarfladim", "сарфладим", "berdim", "бердим",
		"за", "на", "для", "и", "потратил", "потратила", "заплатил", "заплатила", "купил", "купила",
		"for", "on", "and", "spent", "paid", "bought",
	} {
		filler[w] = true
	}
}

// filler words carry no meaning for a description: "taksi uchun" → "taksi".
// "on" is also Uzbek 10 (o'n without apostrophe), so filler is checked separately from vocab.
var filler = map[string]bool{}

// add registers w, plus its spelling without apostrophes (o'n → on) unless that is taken.
func add(w string, e entry) {
	vocab[w] = e
	if bare := strings.ReplaceAll(w, "'", ""); bare != w {
		if _, taken := vocab[bare]; !taken {
			vocab[bare] = e
		}
	}
}

// Uzbek case and derivation suffixes that may stick to scale and currency words:
// mingga, so'mlik, dollarga, мингдан.
var suffixes = []string{"ning", "lik", "dan", "ga", "ka", "qa", "ni", "нинг", "лик", "дан", "га", "ка", "қа", "ни"}

func lookup(w string) (entry, bool) {
	if e, ok := vocab[w]; ok {
		return e, true
	}
	for _, suf := range suffixes {
		stem, ok := strings.CutSuffix(w, suf)
		if !ok || stem == "" {
			continue
		}
		if e, ok := vocab[stem]; ok && !e.needsNum && (e.kind == kScale || e.kind == kCurrency) {
			return e, true
		}
	}
	// полмиллиона, полтысячи
	if rest, ok := strings.CutPrefix(w, "пол"); ok {
		if e, ok := vocab[rest]; ok && e.kind == kScale && !e.needsNum {
			return entry{kind: kScale, scale: e.scale, unit: big.NewRat(1, 2)}, true
		}
	}
	return entry{}, false
}
