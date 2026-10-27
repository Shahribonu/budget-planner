package api

import (
	"net/http"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
)

// ---- debts ----

type debtJSON struct {
	ID             int64     `json:"id"`
	Direction      string    `json:"direction"`
	Counterparty   string    `json:"counterparty"`
	Currency       string    `json:"currency"`
	PrincipalMinor int64     `json:"principal_minor"`
	RemainingMinor int64     `json:"remaining_minor"`
	DueOn          *date     `json:"due_on"`
	Note           string    `json:"note"`
	Closed         bool      `json:"closed"`
	CreatedAt      time.Time `json:"created_at"`
}

func optDate(t *time.Time) *date {
	if t == nil {
		return nil
	}
	return &date{*t}
}

func toDebtJSON(d ledger.Debt) debtJSON {
	return debtJSON{d.ID, d.Direction, d.Counterparty, d.Currency, d.Principal, d.Remaining, optDate(d.DueOn), d.Note, d.Remaining == 0, d.CreatedAt}
}

func (s *Server) listDebts(w http.ResponseWriter, r *http.Request) {
	ds, err := s.ledger.ListDebts(r.Context(), userID(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]debtJSON, len(ds))
	for i, d := range ds {
		out[i] = toDebtJSON(d)
	}
	writeJSON(w, http.StatusOK, map[string]any{"debts": out})
}

func (s *Server) createDebt(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Direction    string `json:"direction"`
		Counterparty string `json:"counterparty"`
		Currency     string `json:"currency"`
		AmountMinor  int64  `json:"amount_minor"`
		AccountID    *int64 `json:"account_id"`
		OccurredOn   date   `json:"occurred_on"`
		DueOn        *date  `json:"due_on"`
		Note         string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	var due *time.Time
	if in.DueOn != nil {
		due = &in.DueOn.Time
	}
	d, err := s.ledger.CreateDebt(r.Context(), userID(r), ledger.NewDebt{
		Direction: in.Direction, Counterparty: in.Counterparty, Currency: in.Currency, AmountMinor: in.AmountMinor,
		AccountID: in.AccountID, OccurredOn: in.OccurredOn.Time, DueOn: due, Note: in.Note,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toDebtJSON(d))
}

func (s *Server) getDebt(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	d, history, err := s.ledger.GetDebt(r.Context(), userID(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"debt": toDebtJSON(d), "transactions": toTransactionsJSON(history)})
}

func (s *Server) patchDebt(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	// due_on: absent keeps it, null clears it, a date sets it.
	var raw map[string]any
	var in struct {
		Counterparty *string `json:"counterparty"`
		Note         *string `json:"note"`
		DueOn        *date   `json:"due_on"`
	}
	if !decodeTwice(w, r, &in, &raw) {
		return
	}
	p := ledger.DebtPatch{Counterparty: in.Counterparty, Note: in.Note}
	if _, ok := raw["due_on"]; ok {
		p.SetDue = true
		if in.DueOn != nil {
			p.DueOn = &in.DueOn.Time
		}
	}
	d, err := s.ledger.UpdateDebt(r.Context(), userID(r), id, p)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDebtJSON(d))
}

func (s *Server) deleteDebt(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.ledger.DeleteDebt(r.Context(), userID(r), id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type paymentInput struct {
	AmountMinor int64 `json:"amount_minor"`
	AccountID   int64 `json:"account_id"`
	OccurredOn  date  `json:"occurred_on"`
}

func (s *Server) payDebt(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in paymentInput
	if !decode(w, r, &in) {
		return
	}
	t, err := s.ledger.RecordDebtPayment(r.Context(), userID(r), id, in.AmountMinor, in.AccountID, in.OccurredOn.Time)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toTransactionJSON(t))
}

// ---- installment plans ----

type scheduleJSON struct {
	N           int32 `json:"n"`
	DueOn       date  `json:"due_on"`
	AmountMinor int64 `json:"amount_minor"`
	Paid        bool  `json:"paid"`
}

type planJSON struct {
	ID               int64          `json:"id"`
	Title            string         `json:"title"`
	Provider         string         `json:"provider"`
	CategoryID       int64          `json:"category_id"`
	Currency         string         `json:"currency"`
	TotalMinor       int64          `json:"total_minor"`
	PaidMinor        int64          `json:"paid_minor"`
	RemainingMinor   int64          `json:"remaining_minor"`
	Months           int32          `json:"months"`
	PayFromAccountID *int64         `json:"pay_from_account_id"`
	Next             *scheduleJSON  `json:"next"`
	Schedule         []scheduleJSON `json:"schedule"`
	Closed           bool           `json:"closed"`
	CreatedAt        time.Time      `json:"created_at"`
}

func toPlanJSON(p ledger.Plan) planJSON {
	out := planJSON{
		ID: p.ID, Title: p.Title, Provider: p.Provider, CategoryID: p.CategoryID, Currency: p.Currency,
		TotalMinor: p.Total, PaidMinor: p.Paid, RemainingMinor: p.Remaining, Months: p.Months,
		PayFromAccountID: p.PayFromAccountID, Closed: p.Remaining == 0, CreatedAt: p.CreatedAt,
		Schedule: make([]scheduleJSON, len(p.Schedule)),
	}
	for i, r := range p.Schedule {
		out.Schedule[i] = scheduleJSON{r.N, date{r.DueOn}, r.Amount, r.Paid}
	}
	if p.Next != nil {
		out.Next = &scheduleJSON{p.Next.N, date{p.Next.DueOn}, p.Next.Amount, false}
	}
	return out
}

func (s *Server) listPlans(w http.ResponseWriter, r *http.Request) {
	ps, err := s.ledger.ListPlans(r.Context(), userID(r))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]planJSON, len(ps))
	for i, p := range ps {
		out[i] = toPlanJSON(p)
	}
	writeJSON(w, http.StatusOK, map[string]any{"installments": out})
}

func (s *Server) createPlan(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title            string `json:"title"`
		Provider         string `json:"provider"`
		CategoryID       int64  `json:"category_id"`
		Currency         string `json:"currency"`
		TotalMinor       int64  `json:"total_minor"`
		Months           int    `json:"months"`
		FirstDueOn       date   `json:"first_due_on"`
		PayFromAccountID *int64 `json:"pay_from_account_id"`
		OccurredOn       date   `json:"occurred_on"`
		RecordPurchase   *bool  `json:"record_purchase"`
	}
	if !decode(w, r, &in) {
		return
	}
	record := in.RecordPurchase == nil || *in.RecordPurchase
	p, err := s.ledger.CreatePlan(r.Context(), userID(r), ledger.NewPlan{
		Title: in.Title, Provider: in.Provider, CategoryID: in.CategoryID, Currency: in.Currency, TotalMinor: in.TotalMinor,
		Months: in.Months, FirstDueOn: in.FirstDueOn.Time, PayFromAccountID: in.PayFromAccountID,
		OccurredOn: in.OccurredOn.Time, RecordPurchase: record,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPlanJSON(p))
}

func (s *Server) getPlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	p, err := s.ledger.GetPlan(r.Context(), userID(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlanJSON(p))
}

func (s *Server) deletePlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.ledger.DeletePlan(r.Context(), userID(r), id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) payPlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in paymentInput
	if !decode(w, r, &in) {
		return
	}
	t, err := s.ledger.PayInstallment(r.Context(), userID(r), id, in.AmountMinor, in.AccountID, in.OccurredOn.Time)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toTransactionJSON(t))
}
