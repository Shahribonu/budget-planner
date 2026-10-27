package i18n

import "testing"

func TestMoney(t *testing.T) {
	tests := []struct {
		lang     string
		minor    int64
		currency string
		want     string
	}{
		{"uz-Latn", 2500000, "UZS", "25 000 so'm"},
		{"ru", 1200000000, "UZS", "12 000 000 сум"},
		{"uz-Cyrl", 100, "UZS", "1 сўм"},
		{"en", 1250, "USD", "$12.50"},
		{"ru", 1250, "USD", "$12,50"},
		{"en", 2000, "USD", "$20"},
		{"uz-Latn", -4500000, "UZS", "−45 000 so'm"},
	}
	for _, tt := range tests {
		if got := Money(tt.lang, tt.minor, tt.currency); got != tt.want {
			t.Errorf("Money(%s, %d, %s) = %q, want %q", tt.lang, tt.minor, tt.currency, got, tt.want)
		}
	}
}

// Every language has every key the default language has.
func TestCatalogComplete(t *testing.T) {
	for lang, msgs := range catalog {
		for key := range catalog[Default] {
			if _, ok := msgs[key]; !ok {
				t.Errorf("%s is missing %q", lang, key)
			}
		}
	}
}

func TestFallback(t *testing.T) {
	if got := T("xx", "btn.undo"); got != catalog[Default]["btn.undo"] {
		t.Errorf("unknown language fallback = %q", got)
	}
	if got := T("ru", "no.such.key"); got != "no.such.key" {
		t.Errorf("missing key = %q", got)
	}
	if got := Category("ru", "taxi"); got != "Такси" {
		t.Errorf("Category = %q", got)
	}
}
