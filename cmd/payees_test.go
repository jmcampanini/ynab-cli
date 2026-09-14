package cmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestPayeesListNamesTransferAccounts(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "payees", "list")
	if err != nil {
		t.Fatal(err)
	}

	want := `NAME                       TRANSFER ACCOUNT
Landlord
Costco
Amazon
Unknown Vendor
New Shop
Employer
Old Bakery
Insurance Co
Transfer : Chase Checking  Chase Checking
Transfer : Visa            Visa
10 payees
`
	if out != want {
		t.Errorf("payees list =\n%s\nwant\n%s", out, want)
	}
	if want := []string{"/plans", "/plans/p1/accounts", "/plans/p1/payees"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v", h.requests, want)
	}
}

func TestPayeesListUnusedFetchesTheWholeRegister(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "payees", "list", "--unused", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	want := `{"id":"p7","name":"Old Bakery"}
{"id":"p8","name":"Insurance Co"}
`
	if out != want {
		t.Errorf("payees list --unused --jsonl =\n%s\nwant\n%s", out, want)
	}
	if last := h.requests[len(h.requests)-1]; last != "/plans/p1/transactions?since_date=2024-01-01" {
		t.Errorf("register request = %q, want the plan's first month as since_date", last)
	}
}

func TestPayeesGetResolvesNames(t *testing.T) {
	h := newHarness(t)

	byName, err := h.execute(t, "payees", "get", "transfer : visa")
	if err != nil {
		t.Fatal(err)
	}
	byID, err := h.execute(t, "payees", "get", "p2", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	_, missing := h.execute(t, "payees", "get", "Costcos")

	want := `id                   tp2
name                 Transfer : Visa
transfer account     Visa
transfer account id  a2
`
	if byName != want {
		t.Errorf("payees get =\n%s\nwant\n%s", byName, want)
	}
	if byID != "{\"id\":\"p2\",\"name\":\"Costco\"}\n" {
		t.Errorf("payees get p2 --jsonl = %s", byID)
	}
	if missing == nil || ExitCode(missing) != ExitFailure || !strings.HasPrefix(missing.Error(), `payee "Costcos" not found; closest names: "Costco"`) {
		t.Errorf("payees get Costcos = %v", missing)
	}
}
