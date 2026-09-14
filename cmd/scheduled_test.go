package cmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestScheduledListRendersSplits(t *testing.T) {
	h := newHarness(t)

	table, err := h.execute(t, "scheduled", "list")
	if err != nil {
		t.Fatal(err)
	}
	csv, err := h.execute(t, "scheduled", "list", "--csv")
	if err != nil {
		t.Fatal(err)
	}

	wantTable := `NEXT DATE   FREQUENCY  ACCOUNT         PAYEE         CATEGORY      MEMO        AMOUNT
2026-10-01  monthly    Chase Checking  Landlord      Rent                  -$1,500.00
2027-01-15  yearly     Chase Checking  Insurance Co  Split         annual    -$600.00
                                                       Dining Out            -$100.00
                                                       Internet    modem     -$500.00
2 scheduled transactions
`
	if table != wantTable {
		t.Errorf("scheduled list =\n%s\nwant\n%s", table, wantTable)
	}
	wantCSV := `id,parent_id,next_date,first_date,frequency,account,account_id,payee,payee_id,category,category_id,memo,amount,flag_color,flag_name,transfer_account,transfer_account_id
st1,,2026-10-01,2024-02-01,monthly,Chase Checking,a1,Landlord,p1,Rent,c2,,-1500.00,,,,
st2,,2027-01-15,2027-01-15,yearly,Chase Checking,a1,Insurance Co,p8,Split,,annual,-600.00,blue,,,
ss1,st2,2027-01-15,2027-01-15,yearly,Chase Checking,a1,,,Dining Out,c3,,-100.00,blue,,,
ss2,st2,2027-01-15,2027-01-15,yearly,Chase Checking,a1,,,Internet,c1,modem,-500.00,blue,,,
`
	if csv != wantCSV {
		t.Errorf("scheduled list --csv =\n%s\nwant\n%s", csv, wantCSV)
	}
	if want := []string{"/plans", "/plans/p1/accounts", "/plans/p1/scheduled_transactions"}; !reflect.DeepEqual(h.requests[:3], want) {
		t.Errorf("requests = %v, want %v", h.requests[:3], want)
	}
}

func TestScheduledGetShowsLines(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "scheduled", "get", "st2")
	if err != nil {
		t.Fatal(err)
	}
	jsonl, err := h.execute(t, "scheduled", "get", "st1", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	_, missing := h.execute(t, "scheduled", "get", "nope")

	want := `id                st2
next date         2027-01-15
first date        2027-01-15
frequency         yearly
account           Chase Checking
payee             Insurance Co
category          Split
memo              annual
amount            -$600.00
flag              blue
transfer account

PAYEE  CATEGORY    MEMO     AMOUNT
       Dining Out         -$100.00
       Internet    modem  -$500.00
`
	if out != want {
		t.Errorf("scheduled get st2 =\n%s\nwant\n%s", out, want)
	}
	wantJSONL := `{"id":"st1","next_date":"2026-10-01","first_date":"2024-02-01","frequency":"monthly","account":"Chase Checking","account_id":"a1","payee":"Landlord","payee_id":"p1","category":"Rent","category_id":"c2","amount":-1500.00}` + "\n"
	if jsonl != wantJSONL {
		t.Errorf("scheduled get st1 --jsonl =\n%s\nwant\n%s", jsonl, wantJSONL)
	}
	if missing == nil || ExitCode(missing) != ExitFailure || !strings.Contains(missing.Error(), "not found") {
		t.Errorf("scheduled get nope = %v", missing)
	}
}
