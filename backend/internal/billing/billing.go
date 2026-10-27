// Package billing sells Premium for Telegram Stars and answers "is this user Premium?".
// Entitlement is one date, users.premium_until; payments only ever extend it.
package billing

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/telegram"
)

const (
	Monthly = "monthly"
	Yearly  = "yearly"

	subscriptionPeriod = 30 * 24 * 3600 // the only period Telegram allows for Stars subscriptions
	TrialAfterDays     = 7              // logged days before the free trial
)

type Prices struct {
	MonthlyStars int
	YearlyStars  int
}

type Service struct {
	q      *dbq.Queries
	tg     *telegram.Client
	prices Prices
	now    func() time.Time
}

func New(pool *pgxpool.Pool, tg *telegram.Client, prices Prices) *Service {
	return &Service{q: dbq.New(pool), tg: tg, prices: prices, now: time.Now}
}

func (s *Service) Prices() Prices { return s.prices }

// Active reports whether a user has Premium now.
func Active(u dbq.User, now time.Time) bool {
	return u.PremiumUntil != nil && u.PremiumUntil.After(now)
}

func payload(plan string, userID int64) string { return fmt.Sprintf("premium:%s:%d", plan, userID) }

func parsePayload(p string) (plan string, userID int64, ok bool) {
	parts := strings.Split(p, ":")
	if len(parts) != 3 || parts[0] != "premium" || (parts[1] != Monthly && parts[1] != Yearly) {
		return "", 0, false
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	return parts[1], id, err == nil
}

// InvoiceLink creates a Stars payment link for the Mini App (Telegram.WebApp.openInvoice)
// or a bot button.
func (s *Service) InvoiceLink(ctx context.Context, userID int64, plan, title, description string) (string, error) {
	inv := telegram.Invoice{Title: title, Description: description, Payload: payload(plan, userID), Currency: "XTR"}
	switch plan {
	case Monthly:
		inv.Prices = []telegram.LabeledPrice{{Label: title, Amount: s.prices.MonthlyStars}}
		inv.SubscriptionPeriod = subscriptionPeriod
	case Yearly:
		inv.Prices = []telegram.LabeledPrice{{Label: title, Amount: s.prices.YearlyStars}}
	default:
		return "", fmt.Errorf("unknown plan %q", plan)
	}
	return s.tg.CreateInvoiceLink(ctx, inv)
}

// PreCheckout approves a payment only if it is ours, for this user, at the current price.
func (s *Service) PreCheckout(ctx context.Context, q telegram.PreCheckoutQuery) error {
	plan, userID, ok := parsePayload(q.InvoicePayload)
	want := map[string]int{Monthly: s.prices.MonthlyStars, Yearly: s.prices.YearlyStars}[plan]
	if ok {
		u, err := s.q.GetUser(ctx, userID)
		ok = err == nil && u.TelegramID == q.From.ID && q.Currency == "XTR" && q.TotalAmount == want
	}
	if !ok {
		return s.tg.AnswerPreCheckoutQuery(ctx, q.ID, false, "This offer has changed. Please open it again.")
	}
	return s.tg.AnswerPreCheckoutQuery(ctx, q.ID, true, "")
}

// Paid records a successful payment and extends Premium. It returns the new end date and
// whether this payment was new (Telegram can deliver a payment twice).
func (s *Service) Paid(ctx context.Context, telegramUserID int64, p telegram.SuccessfulPayment) (time.Time, bool, error) {
	plan, userID, ok := parsePayload(p.InvoicePayload)
	if !ok || p.Currency != "XTR" {
		return time.Time{}, false, fmt.Errorf("unexpected payment payload %q", p.InvoicePayload)
	}
	u, err := s.q.GetUser(ctx, userID)
	if err != nil || u.TelegramID != telegramUserID {
		return time.Time{}, false, fmt.Errorf("payment for user %d from telegram user %d", userID, telegramUserID)
	}
	now := s.now()
	until := now.AddDate(0, 0, 30)
	switch {
	case p.SubscriptionExpirationDate > 0:
		until = time.Unix(p.SubscriptionExpirationDate, 0)
	case plan == Yearly:
		base := now
		if Active(u, now) {
			base = *u.PremiumUntil
		}
		until = base.AddDate(1, 0, 0)
	}
	if _, err := s.q.RecordPayment(ctx, dbq.RecordPaymentParams{
		UserID: userID, TelegramChargeID: p.TelegramPaymentChargeID, Payload: p.InvoicePayload,
		Stars: int32(p.TotalAmount), Recurring: p.IsRecurring, PremiumUntil: until,
	}); errors.Is(err, pgx.ErrNoRows) {
		return until, false, nil
	} else if err != nil {
		return time.Time{}, false, err
	}
	got, err := s.q.ExtendPremium(ctx, dbq.ExtendPremiumParams{Until: until, ID: userID})
	if err != nil {
		return time.Time{}, false, err
	}
	return *got, true, nil
}

// MaybeStartTrial gives a one-time 7-day Premium trial once the user has logged on
// TrialAfterDays different days: value first, upgrade prompt after.
func (s *Service) MaybeStartTrial(ctx context.Context, u dbq.User) (time.Time, bool, error) {
	if u.TrialStartedAt != nil || Active(u, s.now()) {
		return time.Time{}, false, nil
	}
	days, err := s.q.LoggedDays(ctx, u.ID)
	if err != nil || days < TrialAfterDays {
		return time.Time{}, false, err
	}
	until, err := s.q.StartTrial(ctx, u.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return *until, true, nil
}
