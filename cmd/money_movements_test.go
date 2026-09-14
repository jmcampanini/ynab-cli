package cmd

import (
	"reflect"
	"testing"
)

func TestMoneyMovementsListNamesCategoriesAndReadyToAssign(t *testing.T) {
	h := newHarness(t)

	table, err := h.execute(t, "money-movements", "list")
	if err != nil {
		t.Fatal(err)
	}
	month, err := h.execute(t, "money-movements", "list", "--month", "current", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	wantTable := `MONTH    MOVED AT             FROM             TO                  AMOUNT  NOTE
                              Rent             Ready to Assign      $5.00
2026-08  2026-08-01 08:00:00  Ready to Assign  Rent             $1,500.00
2026-09  2026-09-03 10:00:00  Dining Out       Internet            $25.00  cover dining
3 money movements
`
	if table != wantTable {
		t.Errorf("money-movements list =\n%s\nwant\n%s", table, wantTable)
	}
	wantMonth := `{"id":"mm1","month":"2026-09","moved_at":"2026-09-03T10:00:00Z","from":"Dining Out","from_id":"c3","to":"Internet","to_id":"c1","amount":25.00,"note":"cover dining","group_id":"mg1"}` + "\n"
	if month != wantMonth {
		t.Errorf("money-movements list --month current --jsonl =\n%s\nwant\n%s", month, wantMonth)
	}
	want := []string{
		"/plans", "/plans/p1/categories", "/plans/p1/money_movements",
		"/plans", "/plans/p1/categories", "/plans/p1/months/2026-09-01/money_movements",
	}
	if !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v", h.requests, want)
	}
}
