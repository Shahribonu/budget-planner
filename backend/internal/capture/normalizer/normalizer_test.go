package normalizer

import (
	"testing"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/money"
)

var (
	tashkent = time.FixedZone("UZT", 5*3600)
	now      = time.Date(2026, 10, 26, 15, 0, 0, 0, tashkent)
	opts     = Options{DefaultCurrency: money.UZS, Now: now}
)

func uzs(n int64) money.Amount { return money.New(n, money.UZS) }
func usd(n int64) money.Amount { return money.New(n, money.USD) }

// Amount grammar from product plan section 6.2 and the variants people actually type.
func TestSingleAmount(t *testing.T) {
	tests := []struct {
		in   string
		want money.Amount
	}{
		// 45 000 in every common spelling
		{"45k", uzs(45000)},
		{"45к", uzs(45000)},
		{"45 k", uzs(45000)},
		{"45 000", uzs(45000)},
		{"45 000", uzs(45000)},
		{"45 000", uzs(45000)},
		{"45.000", uzs(45000)},
		{"45,000", uzs(45000)},
		{"45000", uzs(45000)},
		{"45 ming", uzs(45000)},
		{"45ming", uzs(45000)},
		{"qirq besh ming", uzs(45000)},
		{"Qirq Besh Ming", uzs(45000)},
		{"қирқ беш минг", uzs(45000)},
		{"45 минг", uzs(45000)},
		{"сорок пять тысяч", uzs(45000)},
		{"45 тыщ", uzs(45000)},
		{"45 тысяч", uzs(45000)},
		{"45 тыс", uzs(45000)},
		{"45т", uzs(45000)},
		{"forty five thousand", uzs(45000)},

		// halves
		{"yarim million", uzs(500000)},
		{"ярим миллион", uzs(500000)},
		{"пол миллиона", uzs(500000)},
		{"полмиллиона", uzs(500000)},
		{"0.5 mln", uzs(500000)},
		{"0,5 млн", uzs(500000)},
		{"0.5mln", uzs(500000)},
		{"bir yarim million", uzs(1500000)},
		{"полтора миллиона", uzs(1500000)},
		{"1.5 mln", uzs(1500000)},
		{"1,5 млн", uzs(1500000)},
		{"1 yarim mln", uzs(1500000)},
		{"полтысячи", uzs(500)},
		{"yarim ming", uzs(500)},
		{"45,5 ming", uzs(45500)},

		// compound numbers
		{"2 500 000", uzs(2500000)},
		{"2.500.000", uzs(2500000)},
		{"2,500,000", uzs(2500000)},
		{"1 million 200 ming", uzs(1200000)},
		{"bir million ikki yuz ming", uzs(1200000)},
		{"ikki yuz ellik ming", uzs(250000)},
		{"ikki ming besh yuz", uzs(2500)},
		{"двести пятьдесят тысяч", uzs(250000)},
		{"yuz ming", uzs(100000)},
		{"сто тысяч", uzs(100000)},
		{"o'n besh ming", uzs(15000)},
		{"o‘n besh ming", uzs(15000)},
		{"oʻn besh ming", uzs(15000)},
		{"on besh ming", uzs(15000)},
		{"ўн беш минг", uzs(15000)},
		{"to'qson ming", uzs(90000)},
		{"toqson ming", uzs(90000)},
		{"ming", uzs(1000)},
		{"тысяча", uzs(1000)},
		{"3 mln", uzs(3000000)},
		{"3 лям", uzs(3000000)},
		{"2 mlrd", uzs(2000000000)},
		{"12 500 000 so'm", uzs(12500000)},

		// Uzbek suffixes stuck to scale and currency words
		{"25 mingga", uzs(25000)},
		{"25 minglik", uzs(25000)},
		{"45000so'm", uzs(45000)},
		{"45 000 so'mga", uzs(45000)},
		{"45 000 сўмлик", uzs(45000)},

		// dollars stay dollars, never converted
		{"20 dollar", usd(20)},
		{"$20", usd(20)},
		{"20$", usd(20)},
		{"$ 20", usd(20)},
		{"20 бакс", usd(20)},
		{"20 баксов", usd(20)},
		{"20 usd", usd(20)},
		{"USD 20", usd(20)},
		{"20 долларов", usd(20)},
		{"12.50 usd", money.Amount{Minor: 1250, Currency: money.USD}},
		{"yigirma dollar", usd(20)},
		{"2k usd", usd(2000)},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := Parse(tt.in, opts)
			if len(got) != 1 {
				t.Fatalf("Parse(%q) = %d items %+v, want 1", tt.in, len(got), got)
			}
			if got[0].Amount != tt.want {
				t.Errorf("Parse(%q) amount = %v, want %v", tt.in, got[0].Amount, tt.want)
			}
		})
	}
}

type want struct {
	desc   string
	amount money.Amount
}

func TestItems(t *testing.T) {
	tests := []struct {
		in   string
		want []want
	}{
		{"taksi 25 ming", []want{{"taksi", uzs(25000)}}},
		{"Taksi 25 000 so'm", []want{{"Taksi", uzs(25000)}}},
		{"non 8 ming, sut 14 ming, taksi 20 ming", []want{{"non", uzs(8000)}, {"sut", uzs(14000)}, {"taksi", uzs(20000)}}},
		{"non 8 ming sut 14 ming", []want{{"non", uzs(8000)}, {"sut", uzs(14000)}}},
		{"non 8000, sut 14000", []want{{"non", uzs(8000)}, {"sut", uzs(14000)}}},
		{"non 8 ming\nsut 14 ming", []want{{"non", uzs(8000)}, {"sut", uzs(14000)}}},
		{"25 ming taksiga, 8 ming non", []want{{"taksiga", uzs(25000)}, {"non", uzs(8000)}}},
		{"8 ming 14 ming", []want{{"", uzs(8000)}, {"", uzs(14000)}}},
		{"2 ta non 8 ming", []want{{"2 ta non", uzs(8000)}}},
		{"3 кг яблок 45 тысяч", []want{{"3 кг яблок", uzs(45000)}}},
		{"bir kofe 15 ming", []want{{"bir kofe", uzs(15000)}}},
		{"kofe 15 200", []want{{"kofe", uzs(15200)}}},
		{"taksi uchun 25 ming", []want{{"taksi", uzs(25000)}}},
		{"non oldim 8 ming", []want{{"non", uzs(8000)}}},
		{"spent 25k on taxi", []want{{"taxi", uzs(25000)}}},
		{"25 ming on taxi", []want{{"taxi", uzs(25000)}}},
		{"taksi 25 ming uyga", []want{{"taksi uyga", uzs(25000)}}},
		{"обед 45 тысяч", []want{{"обед", uzs(45000)}}},
		{"заплатил за интернет 100 тысяч", []want{{"интернет", uzs(100000)}}},
		{"$20 for dinner", []want{{"dinner", usd(20)}}},
		{"dinner 20$", []want{{"dinner", usd(20)}}},
		{"kurtka 20 dollar, taksi 30 ming", []want{{"kurtka", usd(20)}, {"taksi", uzs(30000)}}},
		{"rent 3 million, dentist about 500k", []want{{"rent", uzs(3000000)}, {"dentist about", uzs(500000)}}},
		{"ijara 3 mln, tish shifokori 500 ming, qishki kurtka 800 ming", []want{
			{"ijara", uzs(3000000)}, {"tish shifokori", uzs(500000)}, {"qishki kurtka", uzs(800000)},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := Parse(tt.in, opts)
			if len(got) != len(tt.want) {
				t.Fatalf("Parse(%q) = %+v, want %d items", tt.in, got, len(tt.want))
			}
			for i, w := range tt.want {
				if got[i].Description != w.desc || got[i].Amount != w.amount {
					t.Errorf("item %d = {%q %v}, want {%q %v}", i, got[i].Description, got[i].Amount, w.desc, w.amount)
				}
			}
		})
	}
}

func TestNoAmount(t *testing.T) {
	for _, in := range []string{"", "salom", "bir kofe", "сто", "yarim non", "к маме", "2 ta non", "k"} {
		if got := Parse(in, opts); len(got) != 0 {
			t.Errorf("Parse(%q) = %+v, want none", in, got)
		}
	}
}

func TestDate(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, tashkent) }
	tests := []struct {
		in   string
		want time.Time
	}{
		{"taksi 25 ming", day(26)},
		{"bugun taksi 25 ming", day(26)},
		{"kecha taksi 25 ming", day(25)},
		{"кеча такси 25 минг", day(25)},
		{"обед 45 тысяч вчера", day(25)},
		{"yesterday lunch 45k", day(25)},
		{"позавчера такси 30к", day(24)},
	}
	for _, tt := range tests {
		got := Parse(tt.in, opts)
		if len(got) != 1 || !got[0].Date.Equal(tt.want) {
			t.Errorf("Parse(%q) = %+v, want date %v", tt.in, got, tt.want)
		}
	}

	// Date uses the user's time zone: 23:30 in Tashkent is still the same day there.
	late := Options{Now: time.Date(2026, 10, 1, 23, 30, 0, 0, tashkent)}
	if got := Parse("kecha 10k", late); got[0].Date != time.Date(2026, 9, 30, 0, 0, 0, 0, tashkent) {
		t.Errorf("kecha at month start = %v", got[0].Date)
	}
}

func TestFlags(t *testing.T) {
	got := Parse("taksi 25", opts)
	if len(got) != 1 || !got[0].NeedsReview || got[0].Amount != uzs(25) {
		t.Errorf("bare small UZS amount should need review: %+v", got)
	}
	if got := Parse("taksi 25 ming", opts); got[0].NeedsReview {
		t.Errorf("25 ming should not need review")
	}
	if got := Parse("kofe 5 dollar", opts); got[0].NeedsReview || !got[0].CurrencyExplicit {
		t.Errorf("explicit USD: %+v", got[0])
	}
	if got := Parse("kofe 15k", opts); got[0].CurrencyExplicit {
		t.Errorf("default currency should not be explicit")
	}
	if got := Parse("kofe 15", Options{DefaultCurrency: money.USD, Now: now}); got[0].Amount != usd(15) || got[0].NeedsReview {
		t.Errorf("default USD: %+v", got[0])
	}
	if got := Parse("taksi 25 000 so'm", opts); got[0].Source != "25 000 so'm" {
		t.Errorf("Source = %q", got[0].Source)
	}
}
