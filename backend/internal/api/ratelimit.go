package api

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/Shahribonu/budget-planner/backend/internal/auth"
)

// limiter hands out one token bucket per key (user or IP). Idle buckets are dropped.
type limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    rate.Limit
	burst   int
	swept   time.Time
}

type bucket struct {
	lim  *rate.Limiter
	seen time.Time
}

func newLimiter(perSecond float64, burst int) *limiter {
	return &limiter{buckets: map[string]*bucket{}, rate: rate.Limit(perSecond), burst: burst}
}

func (l *limiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.swept) > time.Minute {
		for k, b := range l.buckets {
			if now.Sub(b.seen) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
		l.swept = now
	}
	b := l.buckets[key]
	if b == nil {
		b = &bucket{lim: rate.NewLimiter(l.rate, l.burst)}
		l.buckets[key] = b
	}
	b.seen = now
	return b.lim.AllowN(now, 1)
}

func tooMany(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "1")
	writeError(w, http.StatusTooManyRequests, "rate_limited", "", "too many requests, slow down")
}

// clientIP is the address chi's RealIP middleware settled on.
func clientIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// limitUser throttles each authenticated user.
func (s *Server) limitUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.userLimit.allow(strconv.FormatInt(userID(r), 10)) {
			tooMany(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// userCache remembers which user row a Telegram profile maps to, so requests don't
// upsert the users table every time. A changed name or username misses and upserts.
type userCache struct {
	mu sync.Mutex
	m  map[int64]cachedUser
}

type cachedUser struct {
	id                  int64
	firstName, username string
	until               time.Time
}

func newUserCache() *userCache { return &userCache{m: map[int64]cachedUser{}} }

func (c *userCache) get(tg auth.TelegramUser) (int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[tg.ID]
	if !ok || time.Now().After(e.until) || e.firstName != tg.FirstName || e.username != tg.Username {
		return 0, false
	}
	return e.id, true
}

func (c *userCache) put(tg auth.TelegramUser, id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) > 100_000 {
		c.m = map[int64]cachedUser{}
	}
	c.m[tg.ID] = cachedUser{id, tg.FirstName, tg.Username, time.Now().Add(10 * time.Minute)}
}

// forget drops a user, e.g. after their account was deleted.
func (c *userCache) forget(userID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.m {
		if e.id == userID {
			delete(c.m, k)
		}
	}
}
