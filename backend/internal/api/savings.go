package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
)

// savings reports how much went into and out of savings accounts each month, for the
// last ?months= months (default 12, at most 36) including the current one. Months
// with no movement are left out; balances come from GET /accounts.
func (s *Server) savings(w http.ResponseWriter, r *http.Request) {
	months := 12
	if v := r.URL.Query().Get("months"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 36 {
			writeError(w, http.StatusBadRequest, "invalid", "months", "months must be 1 to 36")
			return
		}
		months = n
	}
	u, err := s.q.GetUser(r.Context(), userID(r))
	if err != nil {
		s.internal(w, r, err)
		return
	}
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		loc = time.UTC
	}
	y, m, _ := s.now().In(loc).Date()
	from := time.Date(y, m-time.Month(months-1), 1, 0, 0, 0, 0, time.UTC)
	rows, err := s.q.SavingsByMonth(r.Context(), dbq.SavingsByMonthParams{UserID: u.ID, FromDate: from})
	if err != nil {
		s.internal(w, r, err)
		return
	}
	type monthJSON struct {
		Month         string `json:"month"`
		Currency      string `json:"currency"`
		PutInMinor    int64  `json:"put_in_minor"`
		TakenOutMinor int64  `json:"taken_out_minor"`
	}
	out := make([]monthJSON, len(rows))
	for i, row := range rows {
		out[i] = monthJSON{row.Month, row.Currency, row.PutIn, row.TakenOut}
	}
	writeJSON(w, http.StatusOK, map[string]any{"current_month": time.Date(y, m, 1, 0, 0, 0, 0, time.UTC).Format("2006-01"), "months": out})
}
