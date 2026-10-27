package categorize

import "testing"

func TestGuess(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"taksi", "taxi"},
		{"taksiga", "taxi"},
		{"Такси до дома", "taxi"},
		{"yandex go", "taxi"},
		{"non", "groceries"},
		{"nonni", "groceries"},
		{"2 ta non", "groceries"},
		{"nonushta", "cafe"},
		{"tushlik", "cafe"},
		{"обед с коллегами", "cafe"},
		{"кофе", "cafe"},
		{"Korzinka", "groceries"},
		{"корзинка", "groceries"},
		{"bozordan meva", "groceries"},
		{"дори дармон", "health"},
		{"аптека", "health"},
		{"tish shifokori", "health"},
		{"qishki kurtka", "clothing"},
		{"ijara", "rent"},
		{"kvartira uchun", "rent"},
		{"paynet", "mobile_internet"},
		{"benzin", "fuel"},
		{"metan", "fuel"},
		{"svet", "utilities"},
		{"to'y uchun sovg'a", "gifts"},
		{"to‘yga sovg‘a", "gifts"},
		{"oylik", "salary"},
		{"oylik keldi", "salary"},
		{"зарплата", "salary"},
		{"keshbek", "cashback"},
		{"kino", "entertainment"},
		{"sartarosh", "beauty"},
		{"jarima", "fees"},
	}
	for _, tt := range tests {
		m, ok := Guess(tt.in)
		if !ok || m.Key != tt.want {
			t.Errorf("Guess(%q) = %q %v, want %q", tt.in, m.Key, ok, tt.want)
		}
	}
	if m, _ := Guess("oylik"); m.Kind != Income {
		t.Errorf("oylik kind = %s", m.Kind)
	}
	if m, _ := Guess("taksi"); m.Kind != Expense {
		t.Errorf("taksi kind = %s", m.Kind)
	}
}

func TestNoGuess(t *testing.T) {
	// Short stems must not match inside unrelated words.
	for _, in := range []string{"", "salom", "uchun", "nonsense", "онлайн", "gulnora", "unknown shop"} {
		if m, ok := Guess(in); ok {
			t.Errorf("Guess(%q) = %q, want none", in, m.Key)
		}
	}
}

func TestPattern(t *testing.T) {
	if got := Pattern("Korzinka 12 Chilonzor"); got != "korzinka chilonzor" {
		t.Errorf("Pattern = %q", got)
	}
	if Pattern("O‘zbekiston") != Pattern("O'zbekiston") {
		t.Error("apostrophes not unified")
	}
}
