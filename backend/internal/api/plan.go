package api

import (
	"net/http"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/planning"
)

type lineJSON struct {
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	AmountMinor int64  `json:"amount_minor"`
	DueOn       *date  `json:"due_on"`
}

type decisionJSON struct {
	Group              planning.Group `json:"group"`
	Reason             string         `json:"reason"`
	SuggestMonth       *string        `json:"suggest_month"`
	MonthlySavingMinor int64          `json:"monthly_saving_minor"`
	Months             int            `json:"months"`
	FromSavingsMinor   int64          `json:"from_savings_minor"`
}

type planItemJSON struct {
	ID             int64         `json:"id"`
	Title          string        `json:"title"`
	CategoryID     *int64        `json:"category_id"`
	AmountMinMinor int64         `json:"amount_min_minor"`
	AmountMaxMinor int64         `json:"amount_max_minor"`
	Priority       string        `json:"priority"`
	DueOn          *date         `json:"due_on"`
	Status         string        `json:"status"`
	PostponedCount int32         `json:"postponed_count"`
	TransactionID  *int64        `json:"transaction_id"`
	Note           string        `json:"note"`
	Decision       *decisionJSON `json:"decision"`
}

func toPlanViewJSON(v planning.View) map[string]any {
	lines := make([]lineJSON, len(v.Inputs.Lines))
	for i, l := range v.Inputs.Lines {
		lines[i] = lineJSON{l.Kind, l.Title, l.Amount, optDate(l.DueOn)}
	}
	items := make([]planItemJSON, len(v.Items))
	for i, it := range v.Items {
		j := planItemJSON{
			ID: it.ID, Title: it.Title, CategoryID: it.CategoryID, AmountMinMinor: it.AmountMin, AmountMaxMinor: it.AmountMax,
			Priority: it.Priority, DueOn: optDate(it.DueOn), Status: it.Status, PostponedCount: it.PostponedCount,
			TransactionID: it.TransactionID, Note: it.Note,
		}
		if d := it.Decision; d != nil {
			dj := &decisionJSON{Group: d.Group, Reason: d.Reason, MonthlySavingMinor: d.MonthlySaving, Months: d.Months, FromSavingsMinor: d.FromSavings}
			if !d.SuggestMonth.IsZero() {
				m := d.SuggestMonth.Format("2006-01")
				dj.SuggestMonth = &m
			}
			j.Decision = dj
		}
		items[i] = j
	}
	in := v.Inputs
	return map[string]any{
		"month":    v.Plan.Month.Format("2006-01"),
		"currency": v.Plan.Currency,
		"inputs": map[string]any{
			"income_minor": in.Income, "income_source": in.IncomeSource, "income_day": in.IncomeDay,
			"reserved_minor": in.Reserved, "lines": lines, "savings_minor": in.Savings,
			"buffer_minor": in.Buffer, "buffer_source": in.BufferSource, "cash_now_minor": in.CashNow,
			"income_override": v.Plan.IncomeOverride != nil, "buffer_override": v.Plan.BufferOverride != nil,
		},
		"result": map[string]any{"free_minor": v.Result.Free, "allocated_minor": v.Result.Allocated, "left_minor": v.Result.Left},
		"items":  items,
	}
}

// planScope reads ?month=YYYY-MM&currency=UZS (defaults: the month being planned, the
// user's main currency).
func (s *Server) planScope(w http.ResponseWriter, r *http.Request) (dbq.User, time.Time, string, bool) {
	u, err := s.q.GetUser(r.Context(), userID(r))
	if err != nil {
		s.internal(w, r, err)
		return u, time.Time{}, "", false
	}
	month, err := s.plan.MonthFor(u, r.URL.Query().Get("month"))
	if err != nil {
		s.fail(w, r, err)
		return u, time.Time{}, "", false
	}
	cur := r.URL.Query().Get("currency")
	if cur == "" {
		cur = u.MainCurrency
	}
	return u, month, cur, true
}

func (s *Server) getPlanView(w http.ResponseWriter, r *http.Request) {
	u, month, cur, ok := s.planScope(w, r)
	if !ok {
		return
	}
	v, err := s.plan.Get(r.Context(), u.ID, month, cur)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlanViewJSON(v))
}

func (s *Server) patchPlanView(w http.ResponseWriter, r *http.Request) {
	u, month, cur, ok := s.planScope(w, r)
	if !ok {
		return
	}
	// income_override, income_day, buffer_override: absent keeps, null clears, a value sets.
	var raw map[string]any
	var in struct {
		IncomeOverride *int64 `json:"income_override"`
		IncomeDay      *int32 `json:"income_day"`
		Savings        *int64 `json:"savings_minor"`
		BufferOverride *int64 `json:"buffer_override"`
	}
	if !decodeTwice(w, r, &in, &raw) {
		return
	}
	_, setIncome := raw["income_override"]
	_, setDay := raw["income_day"]
	_, setBuffer := raw["buffer_override"]
	v, err := s.plan.UpdateSettings(r.Context(), u.ID, month, cur, planning.Settings{
		SetIncome: setIncome, IncomeOverride: in.IncomeOverride, SetIncomeDay: setDay, IncomeDay: in.IncomeDay,
		Savings: in.Savings, SetBuffer: setBuffer, BufferOverride: in.BufferOverride,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlanViewJSON(v))
}

type newItemJSON struct {
	Title      string `json:"title"`
	CategoryID *int64 `json:"category_id"`
	AmountMin  int64  `json:"amount_min_minor"`
	AmountMax  int64  `json:"amount_max_minor"`
	Priority   string `json:"priority"`
	DueOn      *date  `json:"due_on"`
	Note       string `json:"note"`
}

func (s *Server) addPlanItems(w http.ResponseWriter, r *http.Request) {
	u, month, cur, ok := s.planScope(w, r)
	if !ok {
		return
	}
	var in struct {
		Items []newItemJSON `json:"items"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Items) == 0 || len(in.Items) > 50 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_items", "items", "send 1–50 items")
		return
	}
	items := make([]planning.NewItem, len(in.Items))
	for i, it := range in.Items {
		var due *time.Time
		if it.DueOn != nil {
			due = &it.DueOn.Time
		}
		items[i] = planning.NewItem{Title: it.Title, CategoryID: it.CategoryID, AmountMin: it.AmountMin, AmountMax: it.AmountMax, Priority: it.Priority, DueOn: due, Note: it.Note}
	}
	v, err := s.plan.AddItems(r.Context(), u.ID, month, cur, items)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPlanViewJSON(v))
}

// parsePlanText previews items from free text; nothing is saved.
func (s *Server) parsePlanText(w http.ResponseWriter, r *http.Request) {
	u, _, cur, ok := s.planScope(w, r)
	if !ok {
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if !decode(w, r, &in) {
		return
	}
	items, err := s.plan.ParseItems(r.Context(), u.ID, cur, in.Text)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]newItemJSON, len(items))
	for i, it := range items {
		out[i] = newItemJSON{Title: it.Title, CategoryID: it.CategoryID, AmountMin: it.AmountMin, AmountMax: it.AmountMax, Priority: it.Priority, Note: it.Note}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) patchPlanItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var raw map[string]any
	var in struct {
		Title         *string `json:"title"`
		CategoryID    *int64  `json:"category_id"`
		AmountMin     *int64  `json:"amount_min_minor"`
		AmountMax     *int64  `json:"amount_max_minor"`
		Priority      *string `json:"priority"`
		DueOn         *date   `json:"due_on"`
		Status        *string `json:"status"`
		TransactionID *int64  `json:"transaction_id"`
		Note          *string `json:"note"`
	}
	if !decodeTwice(w, r, &in, &raw) {
		return
	}
	p := planning.ItemPatch{Title: in.Title, CategoryID: in.CategoryID, AmountMin: in.AmountMin, AmountMax: in.AmountMax,
		Priority: in.Priority, Status: in.Status, TransactionID: in.TransactionID, Note: in.Note}
	_, p.SetCategory = raw["category_id"]
	_, p.SetDue = raw["due_on"]
	_, p.SetTx = raw["transaction_id"]
	if in.DueOn != nil {
		p.DueOn = &in.DueOn.Time
	}
	v, err := s.plan.UpdateItem(r.Context(), userID(r), id, p)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlanViewJSON(v))
}

func (s *Server) deletePlanItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.plan.DeleteItem(r.Context(), userID(r), id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- recurring ----

type recurringJSON struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	CategoryID  *int64 `json:"category_id"`
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
	DayOfMonth  int32  `json:"day_of_month"`
	AccountID   *int64 `json:"account_id"`
}

func toRecurringJSON(r dbq.Recurring) recurringJSON {
	return recurringJSON{r.ID, r.Kind, r.Title, r.CategoryID, r.Currency, r.Amount, r.DayOfMonth, r.AccountID}
}

func (s *Server) listRecurring(w http.ResponseWriter, r *http.Request) {
	rs, err := s.plan.ListRecurring(r.Context(), userID(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]recurringJSON, len(rs))
	for i, x := range rs {
		out[i] = toRecurringJSON(x)
	}
	writeJSON(w, http.StatusOK, map[string]any{"recurring": out})
}

func (s *Server) createRecurring(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind        string `json:"kind"`
		Title       string `json:"title"`
		CategoryID  *int64 `json:"category_id"`
		Currency    string `json:"currency"`
		AmountMinor int64  `json:"amount_minor"`
		DayOfMonth  int    `json:"day_of_month"`
		AccountID   *int64 `json:"account_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	x, err := s.plan.CreateRecurring(r.Context(), userID(r), planning.NewRecurring{
		Kind: in.Kind, Title: in.Title, CategoryID: in.CategoryID, Currency: in.Currency, Amount: in.AmountMinor, Day: in.DayOfMonth, AccountID: in.AccountID,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toRecurringJSON(x))
}

func (s *Server) patchRecurring(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Title       *string `json:"title"`
		AmountMinor *int64  `json:"amount_minor"`
		DayOfMonth  *int32  `json:"day_of_month"`
	}
	if !decode(w, r, &in) {
		return
	}
	x, err := s.plan.UpdateRecurring(r.Context(), userID(r), id, in.Title, in.AmountMinor, in.DayOfMonth)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toRecurringJSON(x))
}

func (s *Server) deleteRecurring(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.plan.DeleteRecurring(r.Context(), userID(r), id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
