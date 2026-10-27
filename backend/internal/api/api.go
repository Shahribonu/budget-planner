// Package api serves the Mini App HTTP API described in api/openapi.yaml.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shahribonu/budget-planner/backend/internal/auth"
	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/planning"
)

type Config struct {
	BotToken string
	// DevAuth accepts "Authorization: dev <telegram id>" for local development
	// outside Telegram. Never enable in production.
	DevAuth bool
	// TelegramWebhook, when set, is served at POST /telegram/webhook.
	TelegramWebhook http.Handler
}

type Server struct {
	cfg    Config
	pool   *pgxpool.Pool
	q      *dbq.Queries
	ledger *ledger.Service
	plan   *planning.Service
	now    func() time.Time
}

func New(cfg Config, pool *pgxpool.Pool) http.Handler {
	l := ledger.New(pool)
	s := &Server{cfg: cfg, pool: pool, q: dbq.New(pool), ledger: l, plan: planning.NewService(pool, l), now: time.Now}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, requestLog, middleware.Recoverer)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	if cfg.TelegramWebhook != nil {
		r.Method(http.MethodPost, "/telegram/webhook", cfg.TelegramWebhook)
	}

	r.Route("/api", func(r chi.Router) {
		r.Use(limitBody, s.authenticate)
		r.Get("/me", s.getMe)
		r.Patch("/me", s.patchMe)
		r.Get("/accounts", s.listAccounts)
		r.Post("/accounts", s.createAccount)
		r.Patch("/accounts/{id}", s.patchAccount)
		r.Get("/categories", s.listCategories)
		r.Get("/transactions", s.listTransactions)
		r.Post("/transactions", s.createTransaction)
		r.Get("/transactions/{id}", s.getTransaction)
		r.Put("/transactions/{id}", s.putTransaction)
		r.Delete("/transactions/{id}", s.deleteTransaction)
		r.Post("/transactions/{id}/restore", s.restoreTransaction)
		r.Get("/summary", s.summary)
		r.Get("/debts", s.listDebts)
		r.Post("/debts", s.createDebt)
		r.Get("/debts/{id}", s.getDebt)
		r.Patch("/debts/{id}", s.patchDebt)
		r.Delete("/debts/{id}", s.deleteDebt)
		r.Post("/debts/{id}/payments", s.payDebt)
		r.Get("/installments", s.listPlans)
		r.Post("/installments", s.createPlan)
		r.Get("/installments/{id}", s.getPlan)
		r.Delete("/installments/{id}", s.deletePlan)
		r.Post("/installments/{id}/payments", s.payPlan)
		r.Get("/budgets", s.listBudgets)
		r.Post("/budgets", s.createBudget)
		r.Patch("/budgets/{id}", s.patchBudget)
		r.Delete("/budgets/{id}", s.deleteBudget)
		r.Get("/plan", s.getPlanView)
		r.Patch("/plan", s.patchPlanView)
		r.Post("/plan/items", s.addPlanItems)
		r.Post("/plan/parse", s.parsePlanText)
		r.Patch("/plan/items/{id}", s.patchPlanItem)
		r.Delete("/plan/items/{id}", s.deletePlanItem)
		r.Get("/recurring", s.listRecurring)
		r.Post("/recurring", s.createRecurring)
		r.Patch("/recurring/{id}", s.patchRecurring)
		r.Delete("/recurring/{id}", s.deleteRecurring)
	})
	return r
}

// ---- auth ----

type ctxKey struct{}

func userID(r *http.Request) int64 { return r.Context().Value(ctxKey{}).(int64) }

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme, cred, _ := strings.Cut(r.Header.Get("Authorization"), " ")
		var tg auth.TelegramUser
		var err error
		switch {
		case scheme == "tma":
			tg, err = auth.ValidateInitData(cred, s.cfg.BotToken, 24*time.Hour, s.now())
		case scheme == "dev" && s.cfg.DevAuth:
			tg.ID, err = strconv.ParseInt(cred, 10, 64)
			tg.FirstName = "Dev"
		default:
			err = auth.ErrInvalid
		}
		if err != nil || tg.ID <= 0 {
			writeError(w, http.StatusUnauthorized, "unauthorized", "", "invalid or expired Telegram init data")
			return
		}
		var username *string
		if tg.Username != "" {
			username = &tg.Username
		}
		u, err := s.q.UpsertUser(r.Context(), dbq.UpsertUserParams{
			TelegramID: tg.ID, FirstName: tg.FirstName, Username: username, Language: languageFor(tg.LanguageCode),
		})
		if err != nil {
			s.internal(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u.ID)))
	})
}

// languageFor picks the initial app language from Telegram's language_code.
func languageFor(code string) string {
	switch {
	case strings.HasPrefix(code, "ru"):
		return "ru"
	case strings.HasPrefix(code, "en"):
		return "en"
	}
	return "uz-Latn"
}

// ---- helpers ----

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		next.ServeHTTP(w, r)
	})
}

// requestLog logs method, route and status only — never bodies, which hold financial data.
func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		route := r.URL.Path
		if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
			route = rc.RoutePattern()
		}
		slog.Info("http", "method", r.Method, "route", route, "status", ww.Status(),
			"ms", time.Since(start).Milliseconds(), "request_id", middleware.GetReqID(r.Context()))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Field   string `json:"field,omitempty"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code, field, msg string) {
	var b errorBody
	b.Error.Code, b.Error.Field, b.Error.Message = code, field, msg
	writeJSON(w, status, b)
}

func (s *Server) internal(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "err", err, "request_id", middleware.GetReqID(r.Context()))
	writeError(w, http.StatusInternalServerError, "internal", "", "something went wrong")
}

// fail maps domain errors to HTTP responses.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ve *ledger.ValidationError
	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusUnprocessableEntity, ve.Code, ve.Field, ve.Message)
	case errors.Is(err, ledger.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "", "not found")
	default:
		s.internal(w, r, err)
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "", "request body is not valid JSON for this endpoint")
		return false
	}
	return true
}

// decodeTwice decodes the body into v and also into raw, to tell an absent field from null.
func decodeTwice(w http.ResponseWriter, r *http.Request, v any, raw *map[string]any) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "", "request body is not valid JSON for this endpoint")
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if dec.Decode(v) != nil || json.Unmarshal(body, raw) != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "", "request body is not valid JSON for this endpoint")
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "not_found", "", "not found")
		return 0, false
	}
	return id, true
}
