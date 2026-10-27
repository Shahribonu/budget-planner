package planning

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shahribonu/budget-planner/backend/internal/capture/categorize"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/normalizer"
	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/money"
)

var ErrNotFound = ledger.ErrNotFound

func invalid(code, field, msg string) error {
	return &ledger.ValidationError{Code: code, Field: field, Message: msg}
}

type Service struct {
	pool   *pgxpool.Pool
	q      *dbq.Queries
	ledger *ledger.Service
	now    func() time.Time
}

func NewService(pool *pgxpool.Pool, l *ledger.Service) *Service {
	return &Service{pool: pool, q: dbq.New(pool), ledger: l, now: time.Now}
}

// Line is one known obligation counted in Reserved.
type Line struct {
	Kind   string // recurring, installment, debt
	Title  string
	Amount int64
	DueOn  *time.Time
}

type Inputs struct {
	Income       int64
	IncomeSource string // override, recurring, history, none
	IncomeDay    int
	Reserved     int64
	Lines        []Line
	Savings      int64
	Buffer       int64
	BufferSource string // override, history, none
	CashNow      int64
}

type ItemView struct {
	dbq.PlanItem
	Decision *Decision // nil for items that are not planned (bought, postponed, dropped)
}

type View struct {
	Plan   dbq.MonthlyPlan
	Inputs Inputs
	Result Result
	Items  []ItemView
}

func (s *Service) userLoc(u dbq.User) *time.Location {
	if l, err := time.LoadLocation(u.Timezone); err == nil {
		return l
	}
	return time.UTC
}

func (s *Service) today(u dbq.User) time.Time {
	y, m, d := s.now().In(s.userLoc(u)).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// MonthFor parses "YYYY-MM"; empty means the current month, or next month from the 20th on,
// when people plan ahead.
func (s *Service) MonthFor(u dbq.User, month string) (time.Time, error) {
	if month == "" {
		t := s.today(u)
		first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		if t.Day() >= 20 {
			first = first.AddDate(0, 1, 0)
		}
		return first, nil
	}
	m, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, invalid("invalid_month", "month", "month must be YYYY-MM")
	}
	return m, nil
}

func validCurrency(c string) error {
	if c != "UZS" && c != "USD" {
		return invalid("invalid_currency", "currency", "currency must be UZS or USD")
	}
	return nil
}

// ensure returns the month's plan, creating it and carrying over unfinished items from the
// month before on first use.
func (s *Service) ensure(ctx context.Context, userID int64, month time.Time, currency string) (dbq.MonthlyPlan, error) {
	p, err := s.q.GetMonthlyPlan(ctx, dbq.GetMonthlyPlanParams{UserID: userID, Month: month, Currency: currency})
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return p, err
	}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := dbq.New(tx)
		if p, err = q.InsertMonthlyPlan(ctx, dbq.InsertMonthlyPlanParams{UserID: userID, Month: month, Currency: currency}); err != nil {
			return err
		}
		prev, err := q.UnfinishedItemsOfMonth(ctx, dbq.UnfinishedItemsOfMonthParams{UserID: userID, Month: month.AddDate(0, -1, 0), Currency: currency})
		if err != nil {
			return err
		}
		for _, it := range prev {
			id := it.ID
			if _, err := q.InsertPlanItem(ctx, dbq.InsertPlanItemParams{
				PlanID: p.ID, Title: it.Title, CategoryID: it.CategoryID, AmountMin: it.AmountMin, AmountMax: it.AmountMax,
				Priority: it.Priority, PostponedCount: it.PostponedCount + 1, CarriedFrom: &id, Note: it.Note,
			}); err != nil {
				return err
			}
			if err := q.MarkPostponed(ctx, it.ID); err != nil {
				return err
			}
		}
		return nil
	})
	return p, err
}

// Get returns the plan for a month and currency with its inputs and recommendations.
func (s *Service) Get(ctx context.Context, userID int64, month time.Time, currency string) (View, error) {
	if err := validCurrency(currency); err != nil {
		return View{}, err
	}
	p, err := s.ensure(ctx, userID, month, currency)
	if err != nil {
		return View{}, err
	}
	return s.view(ctx, userID, p)
}

func (s *Service) view(ctx context.Context, userID int64, p dbq.MonthlyPlan) (View, error) {
	in, err := s.inputs(ctx, userID, p)
	if err != nil {
		return View{}, err
	}
	items, err := s.q.ListPlanItems(ctx, p.ID)
	if err != nil {
		return View{}, err
	}
	cats, err := s.categoryKeys(ctx, userID)
	if err != nil {
		return View{}, err
	}
	engineItems := []Item{}
	for _, it := range items {
		if it.Status != "planned" {
			continue
		}
		e := Item{ID: it.ID, Amount: it.AmountMax, Priority: Priority(it.Priority), DueOn: it.DueOn, Postponed: int(it.PostponedCount)}
		if it.CategoryID != nil {
			e.Essential = Essential(cats[*it.CategoryID])
		}
		engineItems = append(engineItems, e)
	}
	res := Plan(Input{
		Month: p.Month, Income: in.Income, IncomeDay: in.IncomeDay, CashNow: in.CashNow,
		Reserved: in.Reserved, Savings: in.Savings, Buffer: in.Buffer, Items: engineItems,
		SaveStep: saveStep(p.Currency),
	})
	byID := map[int64]Decision{}
	for _, d := range res.Decisions {
		byID[d.ItemID] = d
	}
	v := View{Plan: p, Inputs: in, Result: res}
	for _, it := range items {
		iv := ItemView{PlanItem: it}
		if d, ok := byID[it.ID]; ok {
			iv.Decision = &d
		}
		v.Items = append(v.Items, iv)
	}
	return v, nil
}

// saveStep rounds suggested monthly savings: 100 000 so'm or 10 dollars.
func saveStep(currency string) int64 {
	if currency == "USD" {
		return 10 * 100
	}
	return 100_000 * 100
}

func (s *Service) categoryKeys(ctx context.Context, userID int64) (map[int64]string, error) {
	cs, err := s.q.ListCategories(ctx, &userID)
	if err != nil {
		return nil, err
	}
	out := map[int64]string{}
	for _, c := range cs {
		if c.Key != nil {
			out[c.ID] = *c.Key
		}
	}
	return out, nil
}

// inputs gathers what the plan's money math needs, each figure from the user's own data
// unless they overrode it.
func (s *Service) inputs(ctx context.Context, userID int64, p dbq.MonthlyPlan) (Inputs, error) {
	in := Inputs{Savings: p.Savings, IncomeDay: 1}
	monthEnd := p.Month.AddDate(0, 1, 0)
	inMonth := func(d time.Time) bool { return !d.Before(p.Month) && d.Before(monthEnd) }

	rec, err := s.q.ListRecurring(ctx, userID)
	if err != nil {
		return in, err
	}
	var recIncome int64
	firstIncomeDay := 0
	for _, r := range rec {
		if r.Currency != p.Currency {
			continue
		}
		if r.Kind == "income" {
			recIncome += r.Amount
			if firstIncomeDay == 0 || int(r.DayOfMonth) < firstIncomeDay {
				firstIncomeDay = int(r.DayOfMonth)
			}
			continue
		}
		due := clampDay(p.Month, int(r.DayOfMonth))
		in.Lines = append(in.Lines, Line{Kind: "recurring", Title: r.Title, Amount: r.Amount, DueOn: &due})
	}

	switch {
	case p.IncomeOverride != nil:
		in.Income, in.IncomeSource = *p.IncomeOverride, "override"
	case recIncome > 0:
		in.Income, in.IncomeSource = recIncome, "recurring"
	default:
		in.Income, in.IncomeSource, err = s.historicIncome(ctx, userID, p)
		if err != nil {
			return in, err
		}
	}
	switch {
	case p.IncomeDay != nil:
		in.IncomeDay = int(*p.IncomeDay)
	case firstIncomeDay > 0:
		in.IncomeDay = firstIncomeDay
	}

	plans, err := s.ledger.ListPlans(ctx, userID)
	if err != nil {
		return in, err
	}
	for _, pl := range plans {
		if pl.Currency != p.Currency || pl.Next == nil {
			continue
		}
		for _, r := range pl.Schedule {
			if r.Paid || !inMonth(r.DueOn) {
				continue
			}
			amount := r.Amount
			if r.N == pl.Next.N {
				amount = pl.Next.Amount // partly paid already
			}
			due := r.DueOn
			title := pl.Title
			if pl.Provider != "" {
				title = pl.Provider + " · " + pl.Title
			}
			in.Lines = append(in.Lines, Line{Kind: "installment", Title: title, Amount: amount, DueOn: &due})
		}
	}
	debts, err := s.ledger.ListDebts(ctx, userID)
	if err != nil {
		return in, err
	}
	for _, d := range debts {
		if d.Direction == "borrowed" && d.Currency == p.Currency && d.Remaining > 0 && d.DueOn != nil && d.DueOn.Before(monthEnd) {
			due := *d.DueOn
			in.Lines = append(in.Lines, Line{Kind: "debt", Title: d.Counterparty, Amount: d.Remaining, DueOn: &due})
		}
	}
	for _, l := range in.Lines {
		in.Reserved += l.Amount
	}

	if p.BufferOverride != nil {
		in.Buffer, in.BufferSource = *p.BufferOverride, "override"
	} else if in.Buffer, in.BufferSource, err = s.historicBuffer(ctx, userID, p); err != nil {
		return in, err
	}

	totals, err := s.q.AssetTotals(ctx, userID)
	if err != nil {
		return in, err
	}
	for _, t := range totals {
		if t.Currency == p.Currency {
			in.CashNow = t.Amount
		}
	}
	return in, nil
}

func clampDay(month time.Time, day int) time.Time {
	last := month.AddDate(0, 1, -1).Day()
	return time.Date(month.Year(), month.Month(), min(day, last), 0, 0, 0, 0, time.UTC)
}

// historicIncome is the lowest of the last three full months that had income: a
// conservative estimate for irregular earners.
func (s *Service) historicIncome(ctx context.Context, userID int64, p dbq.MonthlyPlan) (int64, string, error) {
	u, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return 0, "", err
	}
	t := s.today(u)
	thisMonth := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	rows, err := s.q.IncomeByMonth(ctx, dbq.IncomeByMonthParams{UserID: userID, Currency: p.Currency, FromDate: thisMonth.AddDate(0, -3, 0), ToDate: thisMonth})
	if err != nil {
		return 0, "", err
	}
	if len(rows) == 0 {
		return 0, "none", nil
	}
	low := rows[0].Amount
	for _, r := range rows {
		low = min(low, r.Amount)
	}
	return low, "history", nil
}

// historicBuffer projects everyday spending (groceries, cafes, transport, taxi, fuel) over
// the last 90 days, or since the first transaction if that is later, onto the month.
// Under two weeks of history is too little to guess from.
func (s *Service) historicBuffer(ctx context.Context, userID int64, p dbq.MonthlyPlan) (int64, string, error) {
	u, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return 0, "", err
	}
	today := s.today(u)
	first, err := s.q.FirstTransactionDate(ctx, userID)
	if err != nil {
		return 0, "", err
	}
	from := today.AddDate(0, 0, -90)
	if first.After(from) {
		from = first
	}
	days := int64(today.Sub(from).Hours() / 24)
	if days < 14 {
		return 0, "none", nil
	}
	spent, err := s.q.EverydaySpending(ctx, dbq.EverydaySpendingParams{UserID: userID, Currency: p.Currency, FromDate: from, ToDate: today})
	if err != nil {
		return 0, "", err
	}
	if spent == 0 {
		return 0, "none", nil // history, but none of it everyday spending: no basis for a guess
	}
	daysInMonth := int64(p.Month.AddDate(0, 1, -1).Day())
	step := saveStep(p.Currency) / 10 // round to 10 000 so'm or 1 dollar
	return roundTo(spent*daysInMonth/days, step), "history", nil
}

type Settings struct {
	SetIncome      bool
	IncomeOverride *int64
	SetIncomeDay   bool
	IncomeDay      *int32
	Savings        *int64
	SetBuffer      bool
	BufferOverride *int64
}

func (s *Service) UpdateSettings(ctx context.Context, userID int64, month time.Time, currency string, st Settings) (View, error) {
	if err := validCurrency(currency); err != nil {
		return View{}, err
	}
	for field, v := range map[string]*int64{"income_override": st.IncomeOverride, "savings": st.Savings, "buffer_override": st.BufferOverride} {
		if v != nil && (*v < 0 || *v > 1e15) {
			return View{}, invalid("invalid_amount", field, "amount must not be negative")
		}
	}
	if st.IncomeDay != nil && (*st.IncomeDay < 1 || *st.IncomeDay > 31) {
		return View{}, invalid("invalid_income_day", "income_day", "day must be 1–31")
	}
	p, err := s.ensure(ctx, userID, month, currency)
	if err != nil {
		return View{}, err
	}
	p, err = s.q.UpdateMonthlyPlan(ctx, dbq.UpdateMonthlyPlanParams{
		SetIncome: st.SetIncome, IncomeOverride: st.IncomeOverride, SetIncomeDay: st.SetIncomeDay, IncomeDay: st.IncomeDay,
		Savings: st.Savings, SetBuffer: st.SetBuffer, BufferOverride: st.BufferOverride, ID: p.ID, UserID: userID,
	})
	if err != nil {
		return View{}, err
	}
	return s.view(ctx, userID, p)
}

// ---- items ----

type NewItem struct {
	Title      string
	CategoryID *int64
	AmountMin  int64
	AmountMax  int64 // 0 means the same as AmountMin
	Priority   string
	DueOn      *time.Time
	Note       string
}

func (s *Service) validItem(ctx context.Context, userID int64, it *NewItem) error {
	it.Title = strings.TrimSpace(it.Title)
	if it.Title == "" || utf8.RuneCountInString(it.Title) > 64 {
		return invalid("invalid_title", "title", "title must be 1–64 characters")
	}
	if it.AmountMax == 0 {
		it.AmountMax = it.AmountMin
	}
	if it.AmountMin <= 0 || it.AmountMax < it.AmountMin || it.AmountMax > 1e15 {
		return invalid("invalid_amount", "amount_min", "amount must be positive and min ≤ max")
	}
	switch Priority(it.Priority) {
	case Must, Should, Nice:
	default:
		return invalid("invalid_priority", "priority", "priority must be must, should or nice")
	}
	if it.CategoryID != nil {
		c, err := s.q.GetCategory(ctx, dbq.GetCategoryParams{ID: *it.CategoryID, UserID: &userID})
		if err != nil || c.Kind != "expense" {
			return invalid("invalid_category", "category_id", "unknown expense category")
		}
	}
	if utf8.RuneCountInString(it.Note) > 200 {
		return invalid("invalid_note", "note", "note is too long")
	}
	return nil
}

func (s *Service) AddItems(ctx context.Context, userID int64, month time.Time, currency string, items []NewItem) (View, error) {
	if err := validCurrency(currency); err != nil {
		return View{}, err
	}
	for i := range items {
		if err := s.validItem(ctx, userID, &items[i]); err != nil {
			return View{}, err
		}
	}
	p, err := s.ensure(ctx, userID, month, currency)
	if err != nil {
		return View{}, err
	}
	for _, it := range items {
		if _, err := s.q.InsertPlanItem(ctx, dbq.InsertPlanItemParams{
			PlanID: p.ID, Title: it.Title, CategoryID: it.CategoryID, AmountMin: it.AmountMin, AmountMax: it.AmountMax,
			Priority: it.Priority, DueOn: it.DueOn, Note: it.Note,
		}); err != nil {
			return View{}, err
		}
	}
	return s.view(ctx, userID, p)
}

// Hint words that set priority and are dropped from the title.
var (
	niceHints = []string{"agar bo'lsa", "imkon bo'lsa", "if possible", "если получится", "если будут деньги", "mumkin bo'lsa"}
	mustHints = []string{"albatta", "shart", "обязательно", "must"}
	// "700-900 ming" → a range
	rangeRe = regexp.MustCompile(`(\d+(?:[.,]\d+)?)\s*[-–—]\s*(\d+(?:[.,]\d+)?)(\s*[^\s\d,;.]+)`)
)

// ParseItems turns a message like "rent 3 million, dentist about 500k, winter jacket
// 700-900 ming, AirPods 2.5 million if possible" into plan items with suggested
// categories and priorities. Nothing is saved.
func (s *Service) ParseItems(ctx context.Context, userID int64, currency, text string) ([]NewItem, error) {
	cs, err := s.q.ListCategories(ctx, &userID)
	if err != nil {
		return nil, err
	}
	byKey := map[string]int64{}
	for _, c := range cs {
		if c.Key != nil {
			byKey[*c.Key] = c.ID
		}
	}
	// Mark ranges so the second number can be merged into the first item.
	text = rangeRe.ReplaceAllString(text, "$1$3 ~ $2$3")
	parsed := normalizer.Parse(text, normalizer.Options{DefaultCurrency: money.Currency(currency), Now: s.now()})
	var out []NewItem
	for i := 0; i < len(parsed); i++ {
		it := parsed[i]
		if string(it.Amount.Currency) != currency {
			continue
		}
		title := strings.TrimSpace(strings.Trim(it.Description, "~ "))
		prio := ""
		low := strings.ToLower(title)
		for _, h := range niceHints {
			if strings.Contains(low, h) {
				prio, title = string(Nice), strings.TrimSpace(removeFold(title, h))
			}
		}
		for _, h := range mustHints {
			if strings.Contains(low, h) {
				prio, title = string(Must), strings.TrimSpace(removeFold(title, h))
			}
		}
		for _, filler := range []string{"about", "taxminan", "примерно", "около"} {
			title = strings.TrimSpace(removeFold(title, filler))
		}
		n := NewItem{Title: title, AmountMin: it.Amount.Minor, AmountMax: it.Amount.Minor}
		// "jacket 700 ming ~ 900 ming": the bare second amount closes the range.
		if i+1 < len(parsed) && strings.Trim(parsed[i+1].Description, "~ ") == "" && parsed[i+1].Amount.Minor > n.AmountMin {
			n.AmountMax = parsed[i+1].Amount.Minor
			i++
		}
		if n.Title == "" {
			continue
		}
		key := ""
		if m, ok := categorize.Guess(n.Title); ok && m.Kind == categorize.Expense {
			key = m.Key
			id := byKey[key]
			n.CategoryID = &id
		}
		if prio == "" {
			prio = string(SuggestPriority(key, n.Title))
		}
		n.Priority = prio
		out = append(out, n)
	}
	return out, nil
}

func removeFold(s, sub string) string {
	i := strings.Index(strings.ToLower(s), strings.ToLower(sub))
	if i < 0 {
		return s
	}
	return s[:i] + s[i+len(sub):]
}

type ItemPatch struct {
	Title         *string
	SetCategory   bool
	CategoryID    *int64
	AmountMin     *int64
	AmountMax     *int64
	Priority      *string
	SetDue        bool
	DueOn         *time.Time
	Status        *string
	SetTx         bool
	TransactionID *int64
	Note          *string
}

func (s *Service) UpdateItem(ctx context.Context, userID, id int64, p ItemPatch) (View, error) {
	cur, err := s.q.GetPlanItem(ctx, dbq.GetPlanItemParams{ID: id, UserID: userID})
	if err != nil {
		return View{}, ledger.ErrNotFound
	}
	check := NewItem{Title: cur.Title, CategoryID: cur.CategoryID, AmountMin: cur.AmountMin, AmountMax: cur.AmountMax, Priority: cur.Priority, Note: cur.Note}
	if p.Title != nil {
		check.Title = *p.Title
	}
	if p.SetCategory {
		check.CategoryID = p.CategoryID
	}
	if p.AmountMin != nil {
		check.AmountMin = *p.AmountMin
	}
	if p.AmountMax != nil {
		check.AmountMax = *p.AmountMax
	}
	if p.Priority != nil {
		check.Priority = *p.Priority
	}
	if p.Note != nil {
		check.Note = *p.Note
	}
	if err := s.validItem(ctx, userID, &check); err != nil {
		return View{}, err
	}
	if p.Status != nil && !slices.Contains([]string{"planned", "bought", "postponed", "dropped"}, *p.Status) {
		return View{}, invalid("invalid_status", "status", "unknown status")
	}
	if p.SetTx && p.TransactionID != nil {
		if _, err := s.ledger.Get(ctx, userID, *p.TransactionID); err != nil {
			return View{}, invalid("invalid_transaction", "transaction_id", "unknown transaction")
		}
	}
	title := check.Title
	if _, err := s.q.UpdatePlanItem(ctx, dbq.UpdatePlanItemParams{
		Title: &title, SetCategory: p.SetCategory, CategoryID: p.CategoryID, AmountMin: &check.AmountMin, AmountMax: &check.AmountMax,
		Priority: &check.Priority, SetDue: p.SetDue, DueOn: p.DueOn, Status: p.Status, SetTx: p.SetTx, TransactionID: p.TransactionID,
		Note: p.Note, ID: id, UserID: userID,
	}); err != nil {
		return View{}, err
	}
	plan, err := s.planOf(ctx, userID, cur.PlanID)
	if err != nil {
		return View{}, err
	}
	return s.view(ctx, userID, plan)
}

func (s *Service) planOf(ctx context.Context, userID, planID int64) (dbq.MonthlyPlan, error) {
	return s.q.GetMonthlyPlanByID(ctx, dbq.GetMonthlyPlanByIDParams{ID: planID, UserID: userID})
}

func (s *Service) DeleteItem(ctx context.Context, userID, id int64) error {
	n, err := s.q.DeletePlanItem(ctx, dbq.DeletePlanItemParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ledger.ErrNotFound
	}
	return nil
}

// MatchPurchase finds a planned item a new expense probably is: same month and category,
// amount within 25% of the planned range. The caller asks the user before linking.
func (s *Service) MatchPurchase(ctx context.Context, userID int64, tx ledger.Transaction) (*dbq.PlanItem, error) {
	if tx.Type != ledger.Expense && tx.Type != ledger.InstallmentTx {
		return nil, nil
	}
	items, err := s.q.OpenPlanItemsForMatch(ctx, dbq.OpenPlanItemsForMatchParams{UserID: userID, Currency: tx.Currency, Day: tx.OccurredOn})
	if err != nil {
		return nil, err
	}
	var best *dbq.PlanItem
	var bestDiff int64
	for i, it := range items {
		if tx.CategoryID == nil || it.CategoryID == nil || *it.CategoryID != *tx.CategoryID {
			continue
		}
		if tx.AmountMinor*4 < it.AmountMin*3 || tx.AmountMinor*4 > it.AmountMax*5 {
			continue
		}
		mid := (it.AmountMin + it.AmountMax) / 2
		diff := max(tx.AmountMinor-mid, mid-tx.AmountMinor)
		if best == nil || diff < bestDiff {
			best, bestDiff = &items[i], diff
		}
	}
	return best, nil
}

// MarkBought links a planned item to the transaction that bought it and returns its title.
func (s *Service) MarkBought(ctx context.Context, userID, itemID, txID int64) (string, error) {
	bought := "bought"
	if _, err := s.UpdateItem(ctx, userID, itemID, ItemPatch{Status: &bought, SetTx: true, TransactionID: &txID}); err != nil {
		return "", err
	}
	it, err := s.q.GetPlanItem(ctx, dbq.GetPlanItemParams{ID: itemID, UserID: userID})
	return it.Title, err
}

// ---- recurring income and bills ----

type NewRecurring struct {
	Kind       string
	Title      string
	CategoryID *int64
	Currency   string
	Amount     int64
	Day        int
	AccountID  *int64
}

func (s *Service) ListRecurring(ctx context.Context, userID int64) ([]dbq.Recurring, error) {
	return s.q.ListRecurring(ctx, userID)
}

func (s *Service) CreateRecurring(ctx context.Context, userID int64, in NewRecurring) (dbq.Recurring, error) {
	in.Title = strings.TrimSpace(in.Title)
	switch {
	case in.Kind != "income" && in.Kind != "expense":
		return dbq.Recurring{}, invalid("invalid_kind", "kind", "kind must be income or expense")
	case in.Title == "" || utf8.RuneCountInString(in.Title) > 64:
		return dbq.Recurring{}, invalid("invalid_title", "title", "title must be 1–64 characters")
	case validCurrency(in.Currency) != nil:
		return dbq.Recurring{}, validCurrency(in.Currency)
	case in.Amount <= 0 || in.Amount > 1e15:
		return dbq.Recurring{}, invalid("invalid_amount", "amount_minor", "amount must be positive")
	case in.Day < 1 || in.Day > 31:
		return dbq.Recurring{}, invalid("invalid_day", "day_of_month", "day must be 1–31")
	}
	if in.CategoryID != nil {
		c, err := s.q.GetCategory(ctx, dbq.GetCategoryParams{ID: *in.CategoryID, UserID: &userID})
		if err != nil || c.Kind != in.Kind {
			return dbq.Recurring{}, invalid("invalid_category", "category_id", "unknown category for this kind")
		}
	}
	if in.AccountID != nil {
		if _, err := s.q.GetAssetAccount(ctx, dbq.GetAssetAccountParams{ID: *in.AccountID, UserID: userID}); err != nil {
			return dbq.Recurring{}, invalid("invalid_account", "account_id", "unknown account")
		}
	}
	return s.q.InsertRecurring(ctx, dbq.InsertRecurringParams{
		UserID: userID, Kind: in.Kind, Title: in.Title, CategoryID: in.CategoryID, Currency: in.Currency,
		Amount: in.Amount, DayOfMonth: int32(in.Day), AccountID: in.AccountID,
	})
}

func (s *Service) UpdateRecurring(ctx context.Context, userID, id int64, title *string, amount *int64, day *int32) (dbq.Recurring, error) {
	if title != nil {
		t := strings.TrimSpace(*title)
		if t == "" || utf8.RuneCountInString(t) > 64 {
			return dbq.Recurring{}, invalid("invalid_title", "title", "title must be 1–64 characters")
		}
		title = &t
	}
	if amount != nil && (*amount <= 0 || *amount > 1e15) {
		return dbq.Recurring{}, invalid("invalid_amount", "amount_minor", "amount must be positive")
	}
	if day != nil && (*day < 1 || *day > 31) {
		return dbq.Recurring{}, invalid("invalid_day", "day_of_month", "day must be 1–31")
	}
	r, err := s.q.UpdateRecurring(ctx, dbq.UpdateRecurringParams{Title: title, Amount: amount, DayOfMonth: day, ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ledger.ErrNotFound
	}
	return r, err
}

func (s *Service) DeleteRecurring(ctx context.Context, userID, id int64) error {
	n, err := s.q.SoftDeleteRecurring(ctx, dbq.SoftDeleteRecurringParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ledger.ErrNotFound
	}
	return nil
}
