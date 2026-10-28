package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/i18n"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/telegram"
)

// TermsVersion is the version of the terms and privacy policy shown in the app.
// Raising it asks every user to accept again.
const TermsVersion = 1

const exportCooldownSecs = 10 * 60

// exportData sends the user's data into their bot chat: transactions as CSV and
// everything as JSON. Files go through Telegram because a Mini App can't download them.
func (s *Server) exportData(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Telegram == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "", "export needs the bot")
		return
	}
	ctx, uid := r.Context(), userID(r)
	claim, err := s.q.ClaimExport(ctx, dbq.ClaimExportParams{ID: uid, CooldownSecs: exportCooldownSecs})
	if errors.Is(err, pgx.ErrNoRows) {
		w.Header().Set("Retry-After", strconv.Itoa(exportCooldownSecs))
		writeError(w, http.StatusTooManyRequests, "rate_limited", "", "one export per 10 minutes")
		return
	}
	if err != nil {
		s.internal(w, r, err)
		return
	}
	csvFile, jsonFile, err := s.buildExport(r, uid)
	if err != nil {
		_ = s.q.ReleaseExport(ctx, uid)
		s.fail(w, r, err)
		return
	}
	day := s.now().Format("2006-01-02")
	err = s.cfg.Telegram.SendDocument(ctx, claim.TelegramID, "transactions-"+day+".csv", csvFile, i18n.T(claim.Language, "export.caption"))
	if err == nil {
		err = s.cfg.Telegram.SendDocument(ctx, claim.TelegramID, "budget-planner-"+day+".json", jsonFile, "")
	}
	if err != nil {
		_ = s.q.ReleaseExport(ctx, uid)
		var te *telegram.Error
		if errors.As(err, &te) && (te.Code == 403 || te.Code == 400) {
			writeError(w, http.StatusConflict, "bot_blocked", "", "open the bot and press Start, then try again")
			return
		}
		s.internal(w, r, err)
		return
	}
	_ = s.q.InsertEvent(ctx, dbq.InsertEventParams{UserID: &uid, Name: "data_exported", Props: []byte("{}")})
	writeJSON(w, http.StatusAccepted, map[string]bool{"sent": true})
}

func (s *Server) buildExport(r *http.Request, uid int64) (csvFile, jsonFile []byte, err error) {
	ctx := r.Context()
	u, err := s.q.GetUser(ctx, uid)
	if err != nil {
		return nil, nil, err
	}
	accs, err := s.ledger.ListAccounts(ctx, uid, true)
	if err != nil {
		return nil, nil, err
	}
	cats, err := s.q.ListCategories(ctx, &uid)
	if err != nil {
		return nil, nil, err
	}
	var txs []ledger.Transaction
	for cursor := ""; ; {
		page, next, err := s.ledger.List(ctx, uid, ledger.Filter{Cursor: cursor, Limit: 200})
		if err != nil {
			return nil, nil, err
		}
		txs = append(txs, page...)
		if next == "" {
			break
		}
		cursor = next
	}
	debts, err := s.ledger.ListDebts(ctx, uid)
	if err != nil {
		return nil, nil, err
	}
	plans, err := s.ledger.ListPlans(ctx, uid)
	if err != nil {
		return nil, nil, err
	}
	recurring, err := s.plan.ListRecurring(ctx, uid)
	if err != nil {
		return nil, nil, err
	}

	accountName := map[int64]string{}
	accounts := make([]accountJSON, len(accs))
	for i, a := range accs {
		accountName[a.ID] = a.Name
		accounts[i] = toAccountJSON(a)
	}
	categoryName := map[int64]string{}
	for _, c := range cats {
		switch {
		case c.Name != nil:
			categoryName[c.ID] = *c.Name
		case c.Key != nil:
			categoryName[c.ID] = i18n.Category(u.Language, *c.Key)
		}
	}

	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)
	_ = cw.Write([]string{"date", "type", "amount", "currency", "account", "to_account", "to_amount", "to_currency", "category", "description", "source"})
	for _, t := range txs {
		row := []string{t.OccurredOn.Format("2006-01-02"), string(t.Type), decimal(t.AmountMinor), t.Currency, accountName[t.AccountID], "", "", "", "", t.Description, t.Source}
		if t.ToAccountID != nil {
			row[5] = accountName[*t.ToAccountID]
		}
		if t.ToAmountMinor != nil {
			row[6] = decimal(*t.ToAmountMinor)
		}
		if t.ToCurrency != nil {
			row[7] = *t.ToCurrency
		}
		if t.CategoryID != nil {
			row[8] = categoryName[*t.CategoryID]
		}
		_ = cw.Write(row)
	}
	cw.Flush()

	debtsOut := make([]debtJSON, len(debts))
	for i, d := range debts {
		debtsOut[i] = toDebtJSON(d)
	}
	plansOut := make([]planJSON, len(plans))
	for i, p := range plans {
		plansOut[i] = toPlanJSON(p)
	}
	recurringOut := make([]recurringJSON, len(recurring))
	for i, rc := range recurring {
		recurringOut[i] = toRecurringJSON(rc)
	}
	jsonFile, err = json.MarshalIndent(map[string]any{
		"exported_at": s.now().UTC().Format(time.RFC3339),
		"user": map[string]any{
			"telegram_id": u.TelegramID, "first_name": u.FirstName, "username": u.Username,
			"language": u.Language, "main_currency": u.MainCurrency, "timezone": u.Timezone, "created_at": u.CreatedAt,
		},
		"accounts":     accounts,
		"categories":   categoryName,
		"transactions": toTransactionsJSON(txs),
		"debts":        debtsOut,
		"installments": plansOut,
		"recurring":    recurringOut,
	}, "", "  ")
	return buf.Bytes(), jsonFile, err
}

// decimal formats minor units with two decimals, as both UZS and USD use.
func decimal(minor int64) string {
	sign := ""
	if minor < 0 {
		sign, minor = "-", -minor
	}
	return sign + strconv.FormatInt(minor/100, 10) + "." + strconv.FormatInt(100+minor%100, 10)[1:]
}

// deleteMe erases the user and everything they own. A later app open starts fresh.
func (s *Server) deleteMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirm string `json:"confirm"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Confirm != "delete" {
		writeError(w, http.StatusUnprocessableEntity, "confirmation_required", "confirm", `send {"confirm":"delete"}`)
		return
	}
	err := pgx.BeginFunc(r.Context(), s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteUserDrafts(r.Context(), userID(r)); err != nil {
			return err
		}
		if err := q.DeleteUserTransactions(r.Context(), userID(r)); err != nil {
			return err
		}
		return q.DeleteUser(r.Context(), userID(r))
	})
	if err != nil {
		s.internal(w, r, err)
		return
	}
	s.users.forget(userID(r))
	slog.Info("account deleted") // no ids: the user asked to be forgotten
	w.WriteHeader(http.StatusNoContent)
}
