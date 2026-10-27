package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
)

// ---- me ----

type userJSON struct {
	ID           int64   `json:"id"`
	TelegramID   int64   `json:"telegram_id"`
	FirstName    string  `json:"first_name"`
	Username     *string `json:"username"`
	Language     string  `json:"language"`
	MainCurrency string  `json:"main_currency"`
	Timezone     string  `json:"timezone"`
	Onboarded    bool    `json:"onboarded"`
	// Evening summary in the bot.
	NotifyEnabled bool   `json:"notify_enabled"`
	NotifyTime    string `json:"notify_time"` // HH:MM in Timezone
}

func clock(t pgtype.Time) string {
	min := t.Microseconds / 60e6
	return fmt.Sprintf("%02d:%02d", min/60, min%60)
}

func (s *Server) userResponse(w http.ResponseWriter, r *http.Request, u dbq.User) {
	onboarded, err := s.q.UserHasAssetAccount(r.Context(), u.ID)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userJSON{u.ID, u.TelegramID, u.FirstName, u.Username, u.Language, u.MainCurrency, u.Timezone, onboarded, u.NotifyEnabled, clock(u.NotifyTime)})
}

func (s *Server) getMe(w http.ResponseWriter, r *http.Request) {
	u, err := s.q.GetUser(r.Context(), userID(r))
	if err != nil {
		s.internal(w, r, err)
		return
	}
	_ = s.q.InsertEvent(r.Context(), dbq.InsertEventParams{UserID: &u.ID, Name: "app_open", Props: []byte("{}")})
	s.userResponse(w, r, u)
}

var languages = map[string]bool{"uz-Latn": true, "uz-Cyrl": true, "ru": true, "en": true}

func (s *Server) patchMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Language      *string `json:"language"`
		MainCurrency  *string `json:"main_currency"`
		Timezone      *string `json:"timezone"`
		NotifyEnabled *bool   `json:"notify_enabled"`
		NotifyTime    *string `json:"notify_time"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Language != nil && !languages[*in.Language] {
		writeError(w, http.StatusUnprocessableEntity, "invalid_language", "language", "unsupported language")
		return
	}
	if in.MainCurrency != nil && *in.MainCurrency != "UZS" && *in.MainCurrency != "USD" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_currency", "main_currency", "currency must be UZS or USD")
		return
	}
	if in.Timezone != nil {
		if _, err := time.LoadLocation(*in.Timezone); err != nil || *in.Timezone == "" || *in.Timezone == "Local" {
			writeError(w, http.StatusUnprocessableEntity, "invalid_timezone", "timezone", "unknown time zone")
			return
		}
	}
	var notifyTime pgtype.Time
	if in.NotifyTime != nil {
		t, err := time.Parse("15:04", *in.NotifyTime)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_notify_time", "notify_time", "notify_time must be HH:MM")
			return
		}
		notifyTime = pgtype.Time{Microseconds: int64(t.Hour()*3600+t.Minute()*60) * 1e6, Valid: true}
	}
	u, err := s.q.UpdateUserSettings(r.Context(), dbq.UpdateUserSettingsParams{
		Language: in.Language, MainCurrency: in.MainCurrency, Timezone: in.Timezone, ID: userID(r),
	})
	if err != nil {
		s.internal(w, r, err)
		return
	}
	if in.NotifyEnabled != nil || in.NotifyTime != nil {
		if u, err = s.q.SetNotify(r.Context(), dbq.SetNotifyParams{Enabled: in.NotifyEnabled, NotifyTime: notifyTime, ID: u.ID}); err != nil {
			s.internal(w, r, err)
			return
		}
	}
	s.userResponse(w, r, u)
}

// ---- accounts ----

type accountJSON struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Currency     string `json:"currency"`
	BalanceMinor int64  `json:"balance_minor"`
	Archived     bool   `json:"archived"`
}

func toAccountJSON(a ledger.Account) accountJSON {
	return accountJSON{a.ID, a.Name, a.Type, a.Currency, a.BalanceMinor, a.Archived}
}

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	accs, err := s.ledger.ListAccounts(r.Context(), userID(r), r.URL.Query().Get("include_archived") == "true")
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]accountJSON, len(accs))
	for i, a := range accs {
		out[i] = toAccountJSON(a)
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": out})
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name                string `json:"name"`
		Type                string `json:"type"`
		Currency            string `json:"currency"`
		OpeningBalanceMinor int64  `json:"opening_balance_minor"`
	}
	if !decode(w, r, &in) {
		return
	}
	a, err := s.ledger.CreateAccount(r.Context(), userID(r), ledger.NewAccount(in))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAccountJSON(a))
}

func (s *Server) patchAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Name     *string `json:"name"`
		Archived *bool   `json:"archived"`
	}
	if !decode(w, r, &in) {
		return
	}
	a, err := s.ledger.UpdateAccount(r.Context(), userID(r), id, ledger.AccountPatch(in))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAccountJSON(a))
}

// ---- categories ----

type categoryJSON struct {
	ID   int64   `json:"id"`
	Kind string  `json:"kind"`
	Key  *string `json:"key"`
	Name *string `json:"name"`
	Icon string  `json:"icon"`
}

func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	cs, err := s.q.ListCategories(r.Context(), &uid)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	out := make([]categoryJSON, len(cs))
	for i, c := range cs {
		out[i] = categoryJSON{c.ID, c.Kind, c.Key, c.Name, c.Icon}
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": out})
}

// ---- transactions ----

type date struct{ time.Time }

func (d *date) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	d.Time, err = time.Parse(time.DateOnly, s)
	return err
}

func (d date) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(d.Format(time.DateOnly))), nil
}

type transactionInput struct {
	Type          ledger.Type `json:"type"`
	OccurredOn    date        `json:"occurred_on"`
	AmountMinor   int64       `json:"amount_minor"`
	AccountID     int64       `json:"account_id"`
	ToAccountID   *int64      `json:"to_account_id"`
	ToAmountMinor *int64      `json:"to_amount_minor"`
	CategoryID    *int64      `json:"category_id"`
	Description   string      `json:"description"`
	Source        string      `json:"source"`
	SourceRef     *string     `json:"source_ref"`
}

func (in transactionInput) toLedger() ledger.Input {
	return ledger.Input{
		Type: in.Type, OccurredOn: in.OccurredOn.Time, AmountMinor: in.AmountMinor, AccountID: in.AccountID,
		ToAccountID: in.ToAccountID, ToAmountMinor: in.ToAmountMinor, CategoryID: in.CategoryID,
		Description: in.Description, Source: in.Source, SourceRef: in.SourceRef,
	}
}

type transactionJSON struct {
	ID            int64       `json:"id"`
	Type          ledger.Type `json:"type"`
	OccurredOn    date        `json:"occurred_on"`
	AmountMinor   int64       `json:"amount_minor"`
	Currency      string      `json:"currency"`
	AccountID     *int64      `json:"account_id"` // null for a debt or plan that moved no money
	ToAccountID   *int64      `json:"to_account_id"`
	ToAmountMinor *int64      `json:"to_amount_minor"`
	ToCurrency    *string     `json:"to_currency"`
	CategoryID    *int64      `json:"category_id"`
	Description   string      `json:"description"`
	Source        string      `json:"source"`
	CreatedAt     time.Time   `json:"created_at"`
	DebtID        *int64      `json:"debt_id"`
	InstallmentID *int64      `json:"installment_id"`
}

func toTransactionJSON(t ledger.Transaction) transactionJSON {
	var account *int64
	if t.AccountID != 0 {
		account = &t.AccountID
	}
	return transactionJSON{
		t.ID, t.Type, date{t.OccurredOn}, t.AmountMinor, t.Currency, account, t.ToAccountID,
		t.ToAmountMinor, t.ToCurrency, t.CategoryID, t.Description, t.Source, t.CreatedAt, t.DebtID, t.PlanID,
	}
}

func toTransactionsJSON(ts []ledger.Transaction) []transactionJSON {
	out := make([]transactionJSON, len(ts))
	for i, t := range ts {
		out[i] = toTransactionJSON(t)
	}
	return out
}

func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ledger.Filter{Cursor: q.Get("cursor")}
	var bad string
	parseDate := func(name string) *time.Time {
		v := q.Get(name)
		if v == "" {
			return nil
		}
		d, err := time.Parse(time.DateOnly, v)
		if err != nil {
			bad = name
			return nil
		}
		return &d
	}
	parseID := func(name string) *int64 {
		v := q.Get(name)
		if v == "" {
			return nil
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			bad = name
			return nil
		}
		return &n
	}
	str := func(name string) *string {
		if v := q.Get(name); v != "" {
			return &v
		}
		return nil
	}
	f.From, f.To = parseDate("from"), parseDate("to")
	f.AccountID, f.CategoryID = parseID("account_id"), parseID("category_id")
	f.Type, f.Currency, f.Q = str("type"), str("currency"), str("q")
	if v := q.Get("limit"); v != "" {
		f.Limit, _ = strconv.Atoi(v)
	}
	if bad != "" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_"+bad, bad, "invalid "+bad)
		return
	}

	ts, next, err := s.ledger.List(r.Context(), userID(r), f)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var nextCursor *string
	if next != "" {
		nextCursor = &next
	}
	writeJSON(w, http.StatusOK, map[string]any{"transactions": toTransactionsJSON(ts), "next_cursor": nextCursor})
}

func (s *Server) createTransaction(w http.ResponseWriter, r *http.Request) {
	var in transactionInput
	if !decode(w, r, &in) {
		return
	}
	t, created, err := s.ledger.Record(r.Context(), userID(r), in.toLedger())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, toTransactionJSON(t))
}

func (s *Server) getTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	t, err := s.ledger.Get(r.Context(), userID(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toTransactionJSON(t))
}

func (s *Server) putTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in transactionInput
	if !decode(w, r, &in) {
		return
	}
	li := in.toLedger()
	if li.Source == "" {
		li.Source = "manual"
	}
	t, err := s.ledger.Update(r.Context(), userID(r), id, li)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toTransactionJSON(t))
}

func (s *Server) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.ledger.Delete(r.Context(), userID(r), id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) restoreTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	t, err := s.ledger.Restore(r.Context(), userID(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toTransactionJSON(t))
}

// ---- summary ----

type moneyJSON struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func toMoneyJSON(ms []ledger.Money) []moneyJSON {
	out := make([]moneyJSON, len(ms))
	for i, m := range ms {
		out[i] = moneyJSON{m.AmountMinor, m.Currency}
	}
	return out
}

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	sum, err := s.ledger.Summary(r.Context(), userID(r), r.URL.Query().Get("month"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	type catJSON struct {
		CategoryID  int64  `json:"category_id"`
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
	}
	cats := make([]catJSON, len(sum.ByCategory))
	for i, c := range sum.ByCategory {
		cats[i] = catJSON(c)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"month":       sum.Month,
		"balances":    toMoneyJSON(sum.Balances),
		"income":      toMoneyJSON(sum.Income),
		"expense":     toMoneyJSON(sum.Expense),
		"by_category": cats,
		"recent":      toTransactionsJSON(sum.Recent),
		"owed_to_me":  toMoneyJSON(sum.OwedToMe),
		"i_owe":       toMoneyJSON(sum.IOwe),
	})
}
