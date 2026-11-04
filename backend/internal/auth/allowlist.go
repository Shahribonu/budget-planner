package auth

import (
	"fmt"
	"strconv"
	"strings"
)

// Allowlist limits who may use the bot and the Mini App. A nil or empty list allows
// everyone.
type Allowlist map[int64]bool

// ParseAllowlist reads comma-separated Telegram user ids, as in ALLOWED_TELEGRAM_IDS.
func ParseAllowlist(s string) (Allowlist, error) {
	var a Allowlist
	for _, f := range strings.Split(s, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		id, err := strconv.ParseInt(f, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid Telegram id %q in ALLOWED_TELEGRAM_IDS", f)
		}
		if a == nil {
			a = Allowlist{}
		}
		a[id] = true
	}
	return a, nil
}

func (a Allowlist) Allows(telegramID int64) bool {
	return len(a) == 0 || a[telegramID]
}
