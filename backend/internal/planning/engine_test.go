package planning

import (
	"testing"
	"time"
)

const sum = 100 // minor units per so'm

func day(d int) *time.Time {
	t := time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC)
	return &t
}

var november = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

// The worked example from product plan section 7.4.
func TestPlanSection74Example(t *testing.T) {
	const (
		dentist = iota + 1
		jacket
		gift
		airpods
	)
	res := Plan(Input{
		Month:    november,
		Income:   8_000_000 * sum,
		Reserved: 4_200_000 * sum, // rent, utilities, installment
		Savings:  800_000 * sum,
		Buffer:   2_100_000 * sum,
		CashNow:  8_000_000 * sum,
		SaveStep: 100_000 * sum,
		Items: []Item{
			{ID: dentist, Amount: 500_000 * sum, Priority: Must, Essential: true},
			{ID: jacket, Amount: 800_000 * sum, Priority: Should},
			{ID: gift, Amount: 300_000 * sum, Priority: Should, DueOn: day(18)},
			{ID: airpods, Amount: 2_500_000 * sum, Priority: Nice, Postponed: 2},
		},
	})
	if res.Free != 900_000*sum {
		t.Fatalf("free = %d, want 900 000", res.Free/sum)
	}
	got := map[int64]Decision{}
	for _, d := range res.Decisions {
		got[d.ItemID] = d
	}
	check := func(id int64, g Group, reason string) {
		t.Helper()
		if got[id].Group != g || got[id].Reason != reason {
			t.Errorf("item %d = %s/%s, want %s/%s", id, got[id].Group, got[id].Reason, g, reason)
		}
	}
	check(dentist, Fits, ReasonMust)
	check(gift, Fits, ReasonFixedDate)
	check(jacket, Later, ReasonNotEnough)
	check(airpods, Reconsider, ReasonPostponed)

	if res.Allocated != 800_000*sum || res.Left != 100_000*sum {
		t.Errorf("allocated %d, left %d; want 800 000 and 100 000", res.Allocated/sum, res.Left/sum)
	}
	// "Buy early next month, or this month if you lower savings by 700 000 once."
	if j := got[jacket]; !j.SuggestMonth.Equal(november.AddDate(0, 1, 0)) || j.FromSavings != 700_000*sum {
		t.Errorf("jacket = %+v", j)
	}
	// "Save 500 000 a month (about 5 months)."
	if a := got[airpods]; a.MonthlySaving != 500_000*sum || a.Months != 5 {
		t.Errorf("airpods = %+v", a)
	}
	// Totals reconcile: fitting items sum to free money minus what is left.
	var fits int64
	for _, d := range res.Decisions {
		if d.Group == Fits {
			fits += map[int64]int64{dentist: 500_000, gift: 300_000}[d.ItemID] * sum
		}
	}
	if fits != res.Free-res.Left {
		t.Errorf("fits %d != free − left %d", fits/sum, (res.Free-res.Left)/sum)
	}
}

func TestMustNeverLater(t *testing.T) {
	res := Plan(Input{
		Month: november, Income: 1_000_000 * sum, Reserved: 900_000 * sum, CashNow: 1_000_000 * sum,
		Items: []Item{{ID: 1, Amount: 500_000 * sum, Priority: Must}},
	})
	if d := res.Decisions[0]; d.Group != Fits || d.Reason != ReasonShortfall || res.Left >= 0 {
		t.Errorf("must over budget = %+v, left %d", d, res.Left)
	}
}

// "An item due on the 5th cannot use a salary paid on the 10th."
func TestSalaryTiming(t *testing.T) {
	res := Plan(Input{
		Month: november, Income: 5_000_000 * sum, IncomeDay: 10, CashNow: 200_000 * sum,
		Items: []Item{
			{ID: 1, Amount: 300_000 * sum, Priority: Should, DueOn: day(5)},  // before salary, cash too low
			{ID: 2, Amount: 150_000 * sum, Priority: Should, DueOn: day(6)},  // before salary, cash enough
			{ID: 3, Amount: 300_000 * sum, Priority: Should, DueOn: day(20)}, // after salary
		},
	})
	got := map[int64]Decision{}
	for _, d := range res.Decisions {
		got[d.ItemID] = d
	}
	if got[1].Group != Later || got[1].Reason != ReasonBeforeSalary {
		t.Errorf("due before salary = %+v", got[1])
	}
	if got[2].Group != Fits || got[3].Group != Fits {
		t.Errorf("affordable items = %+v %+v", got[2], got[3])
	}
}

func TestNoFreeMoney(t *testing.T) {
	res := Plan(Input{
		Month: november, Income: 1_000_000 * sum, Reserved: 1_200_000 * sum, SaveStep: 100_000 * sum,
		Items: []Item{
			{ID: 1, Amount: 100_000 * sum, Priority: Should},
			{ID: 2, Amount: 100_000 * sum, Priority: Nice},
		},
	})
	if res.Free >= 0 {
		t.Fatalf("free = %d", res.Free)
	}
	for _, d := range res.Decisions {
		if d.Group == Fits {
			t.Errorf("item %d fits with negative free money", d.ItemID)
		}
		if d.MonthlySaving <= 0 {
			t.Errorf("item %d has no saving suggestion: %+v", d.ItemID, d)
		}
	}
}

func TestScoreOrder(t *testing.T) {
	free := int64(1_000_000 * sum)
	urgentEssential := Score(Item{Amount: 100_000 * sum, Priority: Should, DueOn: day(3), Essential: true}, november, free)
	plainShould := Score(Item{Amount: 100_000 * sum, Priority: Should}, november, free)
	bigShould := Score(Item{Amount: 900_000 * sum, Priority: Should}, november, free)
	nice := Score(Item{Amount: 100_000 * sum, Priority: Nice}, november, free)
	if !(urgentEssential > plainShould && plainShould > bigShould && bigShould > nice) {
		t.Errorf("scores: urgent %v, plain %v, big %v, nice %v", urgentEssential, plainShould, bigShould, nice)
	}
}
