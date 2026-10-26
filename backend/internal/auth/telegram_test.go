package auth

import (
	"encoding/hex"
	"net/url"
	"strconv"
	"testing"
	"time"
)

const botToken = "123456:TEST-token"

// initData builds init data signed the way Telegram signs it.
func initData(token string, authDate time.Time, user string) string {
	v := url.Values{}
	v.Set("auth_date", strconv.FormatInt(authDate.Unix(), 10))
	v.Set("query_id", "AAH")
	v.Set("user", user)
	v.Set("hash", hex.EncodeToString(sign(dataCheckString(v), token)))
	return v.Encode()
}

func TestValidateInitData(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	user := `{"id":42,"first_name":"Aziz","username":"aziz","language_code":"uz"}`
	maxAge := 24 * time.Hour

	u, err := ValidateInitData(initData(botToken, now.Add(-time.Minute), user), botToken, maxAge, now)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 42 || u.FirstName != "Aziz" || u.LanguageCode != "uz" {
		t.Errorf("user = %+v", u)
	}

	tests := []struct {
		name string
		data string
		want error
	}{
		{"other bot's token", initData("999:other", now, user), ErrInvalid},
		{"expired", initData(botToken, now.Add(-25*time.Hour), user), ErrExpired},
		{"no hash", "auth_date=1&user=%7B%7D", ErrInvalid},
		{"garbage", "%%%", ErrInvalid},
		{"missing user id", initData(botToken, now, `{"first_name":"x"}`), ErrInvalid},
	}
	for _, tt := range tests {
		if _, err := ValidateInitData(tt.data, botToken, maxAge, now); err != tt.want {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
	}

	// Tampering with any field breaks the signature.
	v, _ := url.ParseQuery(initData(botToken, now, user))
	v.Set("user", `{"id":43,"first_name":"Aziz"}`)
	if _, err := ValidateInitData(v.Encode(), botToken, maxAge, now); err != ErrInvalid {
		t.Errorf("tampered user: err = %v", err)
	}
}
