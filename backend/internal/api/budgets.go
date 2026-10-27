package api

import (
	"net/http"

	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
)

type budgetJSON struct {
	ID             int64  `json:"id"`
	CategoryID     *int64 `json:"category_id"`
	Currency       string `json:"currency"`
	AmountMinor    int64  `json:"amount_minor"`
	SpentMinor     int64  `json:"spent_minor"`
	RemainingMinor int64  `json:"remaining_minor"` // negative when over
}

func toBudgetJSON(b ledger.Budget) budgetJSON {
	return budgetJSON{b.ID, b.CategoryID, b.Currency, b.Amount, b.Spent, b.Amount - b.Spent}
}

func (s *Server) listBudgets(w http.ResponseWriter, r *http.Request) {
	bs, month, daysLeft, err := s.ledger.Budgets(r.Context(), userID(r), r.URL.Query().Get("month"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]budgetJSON, len(bs))
	for i, b := range bs {
		out[i] = toBudgetJSON(b)
	}
	writeJSON(w, http.StatusOK, map[string]any{"month": month, "days_left": daysLeft, "budgets": out})
}

func (s *Server) createBudget(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CategoryID  *int64 `json:"category_id"`
		Currency    string `json:"currency"`
		AmountMinor int64  `json:"amount_minor"`
	}
	if !decode(w, r, &in) {
		return
	}
	b, err := s.ledger.CreateBudget(r.Context(), userID(r), in.CategoryID, in.Currency, in.AmountMinor)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toBudgetJSON(b))
}

func (s *Server) patchBudget(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		AmountMinor int64 `json:"amount_minor"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.ledger.UpdateBudget(r.Context(), userID(r), id, in.AmountMinor); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteBudget(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.ledger.DeleteBudget(r.Context(), userID(r), id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
