package bank

import (
	"testing"

	"github.com/Shahribonu/budget-planner/backend/internal/money"
)

// Real HUMO Card bot messages (card digits are the user's own, last four only).
var samples = []struct {
	text      string
	kind      Kind
	dir       Direction
	amount    int64
	merchant  string
	at        string
	balance   int64
	cleanName string
}{
	{"🎉 To'ldirish\n➕ 200.000,00 UZS\n📍 Alif uz u2h>Toshkent\n💳 HUMOCARD *5958\n🕓 18:14 23.10.2026\n💰 203.833,04 UZS",
		TopUp, Credit, 20_000_000, "Alif uz u2h>Toshkent", "2026-10-23 18:14", 20_383_304, "Alif U2h"},
	{"💸 To'lov\n➖ 37.470,00 UZS\n📍 AAB CLICK QR OPLATA\n💳 HUMOCARD *5958\n🕓 18:38 23.10.2026\n💰 118.363,04 UZS",
		Payment, Debit, 3_747_000, "AAB CLICK QR OPLATA", "2026-10-23 18:38", 11_836_304, "Click QR"},
	{"💸 To'lov\n➖ 75.200,00 UZS\n📍 IP OOO JETI ASPAN>TA\n💳 HUMOCARD *5958\n🕓 12:03 26.10.2026\n💰 33.163,04 UZS",
		Payment, Debit, 7_520_000, "IP OOO JETI ASPAN>TA", "2026-10-26 12:03", 3_316_304, "Jeti Aspan"},
	{"🎉 To'ldirish\n➕ 18.000,00 UZS\n📍 Alif uz u2h>Toshkent\n💳 HUMOCARD *5958\n🕓 15:48 26.10.2026\n💰 51.163,04 UZS",
		TopUp, Credit, 1_800_000, "Alif uz u2h>Toshkent", "2026-10-26 15:48", 5_116_304, "Alif U2h"},
	{"💸 To'lov\n➖ 2.500,00 UZS\n📍 AAB CLICK OPLATA>Tos\n💳 HUMOCARD *5958\n🕓 08:35 27.10.2026\n💰 46.163,04 UZS",
		Payment, Debit, 250_000, "AAB CLICK OPLATA>Tos", "2026-10-27 08:35", 4_616_304, "Click"},
}

func TestHumoSamples(t *testing.T) {
	for _, s := range samples {
		n, ok := Parse(s.text)
		if !ok {
			t.Errorf("not recognized:\n%s", s.text)
			continue
		}
		at := n.At.Format("2006-01-02 15:04")
		if n.Kind != s.kind || n.Direction != s.dir || n.Amount != (money.Amount{Minor: s.amount, Currency: money.UZS}) ||
			n.Merchant != s.merchant || n.CardMask != "5958" || at != s.at || n.Balance == nil || n.Balance.Minor != s.balance {
			t.Errorf("Parse = %+v (at %s, balance %v)\nwant %+v", n, at, n.Balance, s)
		}
		if got := MerchantName(n.Merchant); got != s.cleanName {
			t.Errorf("MerchantName(%q) = %q, want %q", n.Merchant, got, s.cleanName)
		}
		if _, off := n.At.Zone(); off != 5*3600 {
			t.Errorf("time zone offset = %d", off)
		}
	}
}

func TestVariants(t *testing.T) {
	// Curly apostrophes, the title line with the bot name, plain +/- signs.
	n, ok := Parse("HUMO Card\n💸 To‘lov\n- 1.250,50 UZS\n📍 KORZINKA>Toshkent\n💳 HUMOCARD *0001\n🕓 9:05 01.11.2026\n💰 10,00 UZS")
	if !ok || n.Kind != Payment || n.Amount.Minor != 125_050 || n.CardMask != "0001" || n.At.Hour() != 9 || n.Balance.Minor != 1_000 {
		t.Errorf("variant = %+v ok=%v", n, ok)
	}
	if n, _ := Parse("🔁 O'tkazma\n➕ 50.000,00 UZS\n💳 HUMOCARD *1234\n🕓 10:00 01.11.2026"); n.Kind != TransferIn || n.Balance != nil {
		t.Errorf("transfer in = %+v", n)
	}
	if n, _ := Parse("🏧 Naqd pul yechish\n➖ 500.000,00 UZS\n💳 HUMOCARD *1234\n🕓 10:00 01.11.2026"); n.Kind != Withdrawal {
		t.Errorf("withdrawal = %+v", n)
	}
	if n, _ := Parse("Пополнение\n➕ 1.000,00 UZS\n💳 HUMOCARD *1234"); n.Kind != TopUp || !n.At.IsZero() {
		t.Errorf("russian top-up = %+v", n)
	}
}

func TestNotANotification(t *testing.T) {
	for _, s := range []string{"taksi 25 ming", "HUMOCARD *5958", "➖ 37.470,00 UZS", "💸 To'lov ➖ 37 470 UZS HUMOCARD *5958"} {
		if n, ok := Parse(s); ok {
			t.Errorf("Parse(%q) = %+v", s, n)
		}
	}
}

func TestParseAmount(t *testing.T) {
	for in, want := range map[string]int64{"0,01": 1, "37.470,00": 3_747_000, "1 000 000,99": 100_000_099, "203.833,04": 20_383_304} {
		if got, ok := parseAmount(in); !ok || got != want {
			t.Errorf("parseAmount(%q) = %d %v", in, got, ok)
		}
	}
	for _, in := range []string{"37.470", "37,4", "abc,00"} {
		if _, ok := parseAmount(in); ok {
			t.Errorf("parseAmount(%q) accepted", in)
		}
	}
}
