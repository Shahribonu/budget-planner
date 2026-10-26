// Package auth validates Telegram Mini App init data and resolves the calling user.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalid = errors.New("init data signature invalid")
	ErrExpired = errors.New("init data expired")
)

// TelegramUser is the "user" field of Mini App init data.
type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

// ValidateInitData checks the HMAC signature of Telegram.WebApp.initData as described in
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
// and rejects data older than maxAge.
func ValidateInitData(initData, botToken string, maxAge time.Duration, now time.Time) (TelegramUser, error) {
	vals, err := url.ParseQuery(initData)
	if err != nil {
		return TelegramUser{}, ErrInvalid
	}
	hash := vals.Get("hash")
	if hash == "" {
		return TelegramUser{}, ErrInvalid
	}
	want, err := hex.DecodeString(hash)
	if err != nil {
		return TelegramUser{}, ErrInvalid
	}
	if !hmac.Equal(sign(dataCheckString(vals), botToken), want) {
		return TelegramUser{}, ErrInvalid
	}

	authDate, err := strconv.ParseInt(vals.Get("auth_date"), 10, 64)
	if err != nil {
		return TelegramUser{}, ErrInvalid
	}
	if now.Sub(time.Unix(authDate, 0)) > maxAge {
		return TelegramUser{}, ErrExpired
	}

	var u TelegramUser
	if err := json.Unmarshal([]byte(vals.Get("user")), &u); err != nil || u.ID == 0 {
		return TelegramUser{}, ErrInvalid
	}
	return u, nil
}

// SignInitData adds a valid hash to vals, as Telegram would. For tests and local tooling.
func SignInitData(vals url.Values, botToken string) string {
	vals.Del("hash")
	vals.Set("hash", hex.EncodeToString(sign(dataCheckString(vals), botToken)))
	return vals.Encode()
}

func dataCheckString(vals url.Values) string {
	keys := make([]string, 0, len(vals))
	for k := range vals {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	lines := make([]string, len(keys))
	for i, k := range keys {
		lines[i] = k + "=" + vals.Get(k)
	}
	return strings.Join(lines, "\n")
}

func sign(data, botToken string) []byte {
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
