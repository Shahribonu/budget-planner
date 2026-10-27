// Package planning holds the monthly spending plan's money math. It is deterministic and
// explainable: every recommendation carries the figures it was made from. Models may
// suggest priorities or word the explanation, but never decide what fits.
package planning

import (
	"math"
	"sort"
	"time"
)

type Priority string

const (
	Must   Priority = "must"   // cannot skip: rent, medicine
	Should Priority = "should" // important but flexible: a winter jacket
	Nice   Priority = "nice"   // wants: gadgets, treats
)

type Group string

const (
	Fits       Group = "fits"
	Later      Group = "later"
	Reconsider Group = "reconsider"
)

// Reason codes explain a decision; the UI and bot turn them into sentences.
const (
	ReasonMust         = "must"          // Must items go first
	ReasonFixedDate    = "fixed_date"    // has a date this month and fits
	ReasonFits         = "fits"          // fits in what is left
	ReasonBeforeSalary = "before_salary" // due before salary and not enough money until then
	ReasonNotEnough    = "not_enough"    // not enough left this month
	ReasonTooLarge     = "too_large"     // large compared with monthly free money
	ReasonPostponed    = "postponed"     // postponed several times and large
	ReasonShortfall    = "must_shortfall"
)

type Item struct {
	ID        int64
	Amount    int64 // the upper end of a range: plans should be conservative
	Priority  Priority
	DueOn     *time.Time // must be paid by; nil is flexible
	Essential bool       // health, education, rent, utilities, family
	Postponed int        // times carried over from earlier months
}

type Input struct {
	Month     time.Time // first day of the planned month
	Income    int64     // expected income in the month
	IncomeDay int       // day of month the main income arrives; 0 or 1 = available from the start
	CashNow   int64     // money on hand now, usable before the income arrives
	Reserved  int64     // obligations already known: recurring bills, installments, debts due
	Savings   int64     // planned savings contribution
	Buffer    int64     // everyday spending (groceries, transport, food) for the month
	Items     []Item
	SaveStep  int64 // rounding for suggested monthly savings, e.g. 100 000 so'm in minor units
}

type Decision struct {
	ItemID int64
	Group  Group
	Reason string
	Score  float64
	// For Later and Reconsider: when it becomes affordable and how much to set aside monthly.
	SuggestMonth  time.Time
	MonthlySaving int64
	Months        int
	// For a Should item that misses by less than the savings: lowering savings once makes it fit.
	FromSavings int64
	// For items placed this month: free money left after this item and the ones before it.
	LeftAfter int64
}

type Result struct {
	Income    int64
	Reserved  int64
	Savings   int64
	Buffer    int64
	Free      int64 // Income − Reserved − Savings − Buffer: what plan items can use
	Allocated int64 // sum of items that fit
	Left      int64 // Free − Allocated
	Decisions []Decision
}

// Score ranks flexible items: the user's priority is the strongest signal, then deadline
// urgency, essentialness and how often it was postponed; large items rank lower.
func Score(it Item, monthStart time.Time, free int64) float64 {
	s := 0.0
	switch it.Priority {
	case Must:
		s += 10
	case Should:
		s += 5
	case Nice:
		s += 1
	}
	if it.DueOn != nil {
		days := it.DueOn.Sub(monthStart).Hours() / 24
		switch {
		case days < 10:
			s += 2
		case days < 31:
			s += 1
		}
	}
	if it.Essential {
		s += 1.5
	}
	s += 0.5 * float64(min(it.Postponed, 4))
	if free > 0 {
		s -= math.Min(float64(it.Amount)/float64(free), 3)
	} else {
		s -= 3
	}
	return s
}

// Plan places items, highest score first, until free money runs out.
func Plan(in Input) Result {
	res := Result{Income: in.Income, Reserved: in.Reserved, Savings: in.Savings, Buffer: in.Buffer}
	res.Free = in.Income - in.Reserved - in.Savings - in.Buffer
	left := res.Free
	// Money usable before the income day: cash now, minus obligations, never above free money.
	beforeSalary := min(in.CashNow-in.Reserved, left)
	salaryDay := time.Date(in.Month.Year(), in.Month.Month(), max(in.IncomeDay, 1), 0, 0, 0, 0, time.UTC)

	items := make([]Item, len(in.Items))
	copy(items, in.Items)
	scores := map[int64]float64{}
	for _, it := range items {
		scores[it.ID] = Score(it, in.Month, res.Free)
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if (a.Priority == Must) != (b.Priority == Must) {
			return a.Priority == Must
		}
		if a.Priority == Must && a.DueOn != nil && b.DueOn != nil && !a.DueOn.Equal(*b.DueOn) {
			return a.DueOn.Before(*b.DueOn)
		}
		return scores[a.ID] > scores[b.ID]
	})

	step := max(in.SaveStep, 1)
	monthlySaving := max(roundTo(res.Free/2, step), step)

	for _, it := range items {
		d := Decision{ItemID: it.ID, Score: scores[it.ID]}
		early := in.IncomeDay > 1 && it.DueOn != nil && it.DueOn.Before(salaryDay)
		fitsMoney := it.Amount <= left
		fitsTime := !early || it.Amount <= beforeSalary

		switch {
		case fitsMoney && fitsTime:
			d.Group, d.Reason = Fits, ReasonFits
			if it.Priority == Must {
				d.Reason = ReasonMust
			} else if it.DueOn != nil {
				d.Reason = ReasonFixedDate
			}
			left -= it.Amount
			res.Allocated += it.Amount
			d.LeftAfter = left
			if early {
				beforeSalary -= it.Amount
			}
		case it.Priority == Must:
			// Must items are never moved to later; the plan is short instead.
			d.Group, d.Reason = Fits, ReasonShortfall
			left -= it.Amount
			res.Allocated += it.Amount
			d.LeftAfter = left
		case fitsMoney && !fitsTime:
			d.Group, d.Reason = Later, ReasonBeforeSalary
			d.SuggestMonth = in.Month
		default:
			d.Group, d.Reason = Later, ReasonNotEnough
			large := res.Free <= 0 || it.Amount > res.Free
			if it.Priority == Nice && large {
				d.Group, d.Reason = Reconsider, ReasonTooLarge
				if it.Postponed >= 2 {
					d.Reason = ReasonPostponed
				}
			}
			if !large {
				d.SuggestMonth = in.Month.AddDate(0, 1, 0)
				if short := it.Amount - max(left, 0); it.Priority == Should && short <= in.Savings {
					d.FromSavings = short
				}
			} else {
				d.MonthlySaving = monthlySaving
				d.Months = int((it.Amount + monthlySaving - 1) / monthlySaving)
				d.SuggestMonth = in.Month.AddDate(0, d.Months, 0)
			}
		}
		res.Decisions = append(res.Decisions, d)
	}
	res.Left = left
	return res
}

func roundTo(v, step int64) int64 {
	if v <= 0 {
		return 0
	}
	return (v + step/2) / step * step
}
