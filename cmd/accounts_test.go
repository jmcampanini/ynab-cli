package cmd

import (
	"strings"
	"testing"
)

func TestAccountsListTable(t *testing.T) {
	h := newHarness(t)

	open, err := h.execute(t, "accounts", "list")
	if err != nil {
		t.Fatal(err)
	}
	all, err := h.execute(t, "accounts", "list", "--closed")
	if err != nil {
		t.Fatal(err)
	}

	wantOpen := `NAME            TYPE        KIND    BALANCE    CLEARED  UNCLEARED  IMPORT
Chase Checking  checking    plan  $1,234.56  $1,000.00    $234.56  linked
Visa            creditCard  plan    -$97.81    -$97.81      $0.00  error
2 accounts, 1 closed hidden
`
	if open != wantOpen {
		t.Errorf("accounts list =\n%s\nwant\n%s", open, wantOpen)
	}
	if !strings.Contains(all, "Old Savings (closed)  savings     tracking      $0.00") || !strings.HasSuffix(all, "3 accounts\n") {
		t.Errorf("accounts list --closed =\n%s", all)
	}
	if strings.Contains(open, "\x1b[") {
		t.Error("table carries escape codes with color off")
	}
}

func TestAccountsListMachineFormats(t *testing.T) {
	h := newHarness(t)
	h.terminal = true

	jsonl, err := h.execute(t, "accounts", "list", "--jsonl", "--closed", "--color", "always")
	if err != nil {
		t.Fatal(err)
	}
	csv, err := h.execute(t, "accounts", "list", "--csv", "--closed", "--color", "always")
	if err != nil {
		t.Fatal(err)
	}

	wantJSONL := `{"id":"a1","name":"Chase Checking","type":"checking","on_plan":true,"closed":false,"note":"Main","balance":1234.56,"cleared_balance":1000.00,"uncleared_balance":234.56,"transfer_payee_id":"tp1","direct_import_linked":true,"direct_import_in_error":false,"last_reconciled_at":"2026-08-31T00:00:00Z"}
{"id":"a2","name":"Visa","type":"creditCard","on_plan":true,"closed":false,"balance":-97.81,"cleared_balance":-97.81,"uncleared_balance":0.00,"transfer_payee_id":"tp2","direct_import_linked":true,"direct_import_in_error":true}
{"id":"a3","name":"Old Savings","type":"savings","on_plan":false,"closed":true,"balance":0.00,"cleared_balance":0.00,"uncleared_balance":0.00,"direct_import_linked":false,"direct_import_in_error":false}
`
	if jsonl != wantJSONL {
		t.Errorf("jsonl =\n%s\nwant\n%s", jsonl, wantJSONL)
	}
	wantCSV := `id,name,type,on_plan,closed,note,balance,cleared_balance,uncleared_balance,transfer_payee_id,direct_import_linked,direct_import_in_error,last_reconciled_at
a1,Chase Checking,checking,true,false,Main,1234.56,1000.00,234.56,tp1,true,false,2026-08-31T00:00:00Z
a2,Visa,creditCard,true,false,,-97.81,-97.81,0.00,tp2,true,true,
a3,Old Savings,savings,false,true,,0.00,0.00,0.00,,false,false,
`
	if csv != wantCSV {
		t.Errorf("csv =\n%s\nwant\n%s", csv, wantCSV)
	}
}

func TestAccountsListColor(t *testing.T) {
	h := newHarness(t)
	h.terminal = true

	colored, err := h.execute(t, "accounts", "list", "--closed")
	if err != nil {
		t.Fatal(err)
	}
	h.env["NO_COLOR"] = "1"
	plainOut, err := h.execute(t, "accounts", "list")
	if err != nil {
		t.Fatal(err)
	}
	forced, err := h.execute(t, "accounts", "list", "--color", "always")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(colored, "\x1b[31m-$97.81\x1b[0m") || !strings.Contains(colored, "\x1b[2mOld Savings (closed)\x1b[0m") {
		t.Errorf("terminal output lacks red negatives or faint closed rows:\n%q", colored)
	}
	if strings.Contains(colored, "\x1b[31m$1,234.56") {
		t.Errorf("positive amount is red:\n%q", colored)
	}
	if strings.Contains(plainOut, "\x1b[") {
		t.Errorf("NO_COLOR output carries escapes:\n%q", plainOut)
	}
	if !strings.Contains(forced, "\x1b[31m") {
		t.Errorf("--color always under NO_COLOR lacks escapes:\n%q", forced)
	}
}

func TestAccountsGet(t *testing.T) {
	h := newHarness(t)

	byName, err := h.execute(t, "accounts", "get", "chase checking")
	if err != nil {
		t.Fatal(err)
	}
	byID, err := h.execute(t, "accounts", "get", "a3", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	closed, err := h.execute(t, "accounts", "get", "a3")
	if err != nil {
		t.Fatal(err)
	}

	want := `id                 a1
name               Chase Checking
type               checking
kind               plan
closed             false
note               Main
balance            $1,234.56
cleared            $1,000.00
uncleared          $234.56
transfer payee id  tp1
direct import      linked
last reconciled    2026-08-31 00:00:00
`
	if byName != want {
		t.Errorf("accounts get =\n%s\nwant\n%s", byName, want)
	}
	if !strings.HasPrefix(byID, `{"id":"a3","name":"Old Savings"`) || strings.Count(byID, "\n") != 1 {
		t.Errorf("accounts get a3 --jsonl = %q; closed accounts must resolve", byID)
	}
	if !strings.Contains(closed, "\nnote\nbalance") || !strings.Contains(closed, "\nlast reconciled\n") {
		t.Errorf("empty fields carry trailing spaces:\n%q", closed)
	}
}

func TestAccountsGetMissListsClosestNames(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "accounts", "get", "Chase Cheking")

	if err == nil || out != "" || ExitCode(err) != ExitFailure {
		t.Fatalf("accounts get miss = %q, %v (exit %d)", out, err, ExitCode(err))
	}
	if !strings.Contains(err.Error(), `closest names: "Chase Checking", `) {
		t.Errorf("miss error = %q", err.Error())
	}
}
