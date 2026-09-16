package cmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewWritesExitThreeWithoutAllowWritesAndPreviewWithDryRun(t *testing.T) {
	cases := [][]string{
		{"months", "assign", "Rent", "10"},
		{"months", "move", "10", "--from", "Visa", "--to", "Rent"},
		{"months", "cover", "Dining Out", "--from", "Visa"},
		{"months", "fund", "Wishes: Internet", "--force"},
		{"categories", "create", "Water", "--group", "Bills"},
		{"categories", "update", "Rent", "--note", "due on the 1st"},
		{"category-groups", "create", "Savings"},
		{"category-groups", "rename", "Bills", "Monthly Bills"},
		{"payees", "create", "Corner Store"},
		{"payees", "rename", "Amazon", "Amazon.com"},
		{"accounts", "create", "Emergency Fund", "--type", "savings", "--balance", "0"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			h := newHarness(t)

			out, err := h.execute(t, args...)
			if err == nil || out != "" || ExitCode(err) != ExitWritesDisabled || len(h.requests) != 0 {
				t.Fatalf("execute(%v) = %q, %v (exit %d), requests %v; want exit 3, empty stdout, no request", args, out, err, ExitCode(err), h.requests)
			}

			dry, err := h.execute(t, append(args, "--dry-run", "--jsonl")...)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(dry, `,"dry_run":true}`+"\n") || len(h.writes) != 0 {
				t.Errorf("execute(%v --dry-run --jsonl) = %q, writes %v; want dry_run marked and nothing written", args, dry, h.writes)
			}
		})
	}
}

func TestCategoriesCreateAndUpdateSendTheSaveBody(t *testing.T) {
	cases := []struct {
		args      []string
		wantWrite string
	}{
		{[]string{"create", "Water", "--group", "Bills", "--note", "Utility"}, `POST /plans/p1/categories {"category":{"category_group_id":"g1","name":"Water","note":"Utility"}}`},
		{[]string{"create", "Vacation", "--group", "fun", "--target", "1200", "--target-date", "2027-06"}, `POST /plans/p1/categories {"category":{"category_group_id":"g2","name":"Vacation","goal_target":1200000,"goal_target_date":"2027-06-01"}}`},
		{[]string{"create", "Groceries", "--group", "g2", "--target", "600", "--target-frequency", "monthly", "--needs-whole-amount=false"}, `POST /plans/p1/categories {"category":{"category_group_id":"g2","name":"Groceries","goal_target":600000,"goal_frequency":"monthly","goal_needs_whole_amount":false}}`},
		{[]string{"update", "Rent", "--name", "Housing", "--note", ""}, `PATCH /plans/p1/categories/c2 {"category":{"name":"Housing","note":""}}`},
		{[]string{"update", "Bills: Rent", "--group", "Fun"}, `PATCH /plans/p1/categories/c2 {"category":{"category_group_id":"g2"}}`},
		{[]string{"update", "Rent", "--no-target"}, `PATCH /plans/p1/categories/c2 {"category":{"goal_target":null}}`},
		{[]string{"update", "Rent", "--target-date", "2027-01-15"}, `PATCH /plans/p1/categories/c2 {"category":{"goal_target_date":"2027-01-15"}}`},
		{[]string{"update", "Rent", "--needs-whole-amount"}, `PATCH /plans/p1/categories/c2 {"category":{"goal_needs_whole_amount":true}}`},
		{[]string{"update", "Credit Card Payments: Visa", "--target", "100"}, `PATCH /plans/p1/categories/c6 {"category":{"goal_target":100000}}`},
		{[]string{"update", "Wishes: Internet", "--target", "750"}, `PATCH /plans/p1/categories/c5 {"category":{"goal_target":750000}}`},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			h := newHarness(t)

			_, err := h.execute(t, append([]string{"categories", "--allow-writes"}, tc.args...)...)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(h.writes, []string{tc.wantWrite}) {
				t.Errorf("writes = %v, want [%s]", h.writes, tc.wantWrite)
			}
			if want := []string{"/plans", "/plans/p1/categories"}; !reflect.DeepEqual(h.requests[:2], want) {
				t.Errorf("reads = %v, want %v before the write", h.requests[:2], want)
			}
		})
	}
}

func TestCategoryWritesRefuseWhatTheSpecForbids(t *testing.T) {
	cases := []struct {
		args     []string
		wantErr  string
		wantExit int
		// reads is set when the check needs the plan, such as an amount
		// parsed at the plan's precision.
		reads bool
	}{
		{[]string{"create", "Water", "--group", "Bills", "--target-date", "2027-01"}, "--target-date requires --target on create", ExitUsage, false},
		{[]string{"create", "Water", "--group", "Bills", "--needs-whole-amount"}, "--needs-whole-amount requires --target on create", ExitUsage, false},
		{[]string{"create", "Water", "--group", "Bills", "--target", "10", "--target-frequency", "monthly", "--target-date", "2027-01"}, "target-frequency", ExitUsage, false},
		{[]string{"create", "Water", "--group", "Bills", "--target", "10", "--target-frequency", "daily"}, "invalid --target-frequency", ExitUsage, false},
		{[]string{"create", "Water", "--group", "Bills", "--target", "0"}, "--target must be positive", ExitUsage, true},
		{[]string{"create", "Water", "--group", "Bills", "--target", "5", "--target-date", "2027-13"}, "invalid --target-date", ExitUsage, false},
		{[]string{"create", "", "--group", "Bills"}, "NAME must not be empty", ExitUsage, false},
		{[]string{"update", "Rent", "--target-frequency", "monthly"}, "--target-frequency requires --target", ExitUsage, false},
		{[]string{"update", "Rent", "--no-target", "--target", "5"}, "no-target", ExitUsage, false},
		{[]string{"update", "Rent"}, "at least one field flag", ExitUsage, false},
		{[]string{"update", "Rent", "--name", ""}, "--name must not be empty", ExitUsage, false},
		{[]string{"update", "Credit Card Payments: Visa", "--target", "100", "--target-frequency", "monthly"}, "--target-frequency is not supported on a credit card payment category", ExitFailure, false},
		{[]string{"update", "Credit Card Payments: Visa", "--needs-whole-amount"}, "--needs-whole-amount is not supported on a credit card payment category", ExitFailure, false},
		{[]string{"update", "Inflow: Ready to Assign", "--name", "Income"}, "is internal", ExitFailure, false},
		{[]string{"update", "Rent", "--group", "Internal Master Category"}, `group "Internal Master Category" belongs to the API`, ExitFailure, false},
		{[]string{"update", "Rent", "--name", "internet"}, `group "Bills" already has a category named "Internet"`, ExitFailure, false},
		{[]string{"update", "Rent", "--group", "Wishes", "--name", "Internet"}, `group "Wishes" already has a category named "Internet"`, ExitFailure, false},
		{[]string{"create", "Water", "--group", "Credit Card Payments"}, `group "Credit Card Payments" belongs to the API`, ExitFailure, false},
		{[]string{"create", "rent", "--group", "Bills"}, `group "Bills" already has a category named "Rent"`, ExitFailure, false},
		{[]string{"create", "Water", "--group", "Utilities"}, `category group "Utilities" not found`, ExitFailure, false},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			h := newHarness(t)

			out, err := h.execute(t, append([]string{"categories", "--allow-writes"}, tc.args...)...)

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || ExitCode(err) != tc.wantExit || out != "" {
				t.Errorf("categories %v = %q, %v (exit %d); want exit %d and an error containing %q", tc.args, out, err, ExitCode(err), tc.wantExit, tc.wantErr)
			}
			if len(h.writes) != 0 {
				t.Errorf("writes = %v, want none", h.writes)
			}
			if tc.wantExit == ExitUsage && !tc.reads && len(h.requests) != 0 {
				t.Errorf("usage error made requests: %v", h.requests)
			}
		})
	}
}

func TestCategoryDryRunsPreviewTheTargetFromTheSpecRules(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "categories", "update", "Rent", "--target", "1600", "--target-date", "2027-06", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}

	want := `id                  c2
name                Rent
group               Bills
month               2026-09
hidden              false
internal            false
note
assigned            $1,500.00
activity            -$1,500.00
available           $0.00
target              target balance (TB)
target amount       $1,600.00
target date         2027-06-01
target cadence      monthly
target day          last day of the month
target rollover
target created      2024-01
target progress     100%
target months left  1
target underfunded  $0.00
target funded       $1,500.00
target left         $0.00
target snoozed

dry run, nothing changed: would update category "Bills: Rent"
`
	if out != want {
		t.Errorf("update --dry-run =\n%s\nwant\n%s", out, want)
	}

	created, err := h.execute(t, "categories", "create", "Water", "--group", "Bills", "--target", "40", "--dry-run", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	wantCreated := `{"id":"","name":"Water","group":"Bills","group_id":"g1","month":"2026-09","hidden":false,"internal":false,"assigned":0.00,"activity":0.00,"available":0.00,"target":{"type":"NEED","amount":40.00},"dry_run":true}` + "\n"
	if created != wantCreated {
		t.Errorf("create --dry-run --jsonl =\n%s\nwant\n%s", created, wantCreated)
	}

	card, err := h.execute(t, "categories", "update", "Credit Card Payments: Visa", "--target", "100", "--dry-run", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(card, `"target":{"type":"MF","amount":100.00}`) {
		t.Errorf("credit card update --dry-run --jsonl = %s, want an MF target", card)
	}

	removed, err := h.execute(t, "categories", "update", "Rent", "--no-target", "--dry-run", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(removed, `"target"`) {
		t.Errorf("--no-target --dry-run --jsonl = %s, want no target", removed)
	}
	if len(h.writes) != 0 {
		t.Errorf("dry runs wrote: %v", h.writes)
	}
}

func TestCategoriesUpdatePrintsTheStoredCategory(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "categories", "update", "Rent", "--name", "Housing", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(out, "id                  c2\nname                Housing\ngroup               Bills\n") || !strings.HasSuffix(out, "\nupdated category \"Bills: Rent\"\n") {
		t.Errorf("update =\n%s\nwant the stored record and the summary", out)
	}
}

func TestCategoryGroupsCreateAndRename(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "category-groups", "create", "Savings", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}

	want := `id          gnew
name        Savings
hidden      false
internal    false
categories  0

created category group "Savings"
`
	if out != want {
		t.Errorf("create =\n%s\nwant\n%s", out, want)
	}
	if want := []string{`POST /plans/p1/category_groups {"category_group":{"name":"Savings"}}`}; !reflect.DeepEqual(h.writes, want) {
		t.Errorf("writes = %v, want %v", h.writes, want)
	}

	h = newHarness(t)
	out, err = h.execute(t, "category-groups", "rename", "bills", "Monthly Bills", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}
	// The update answer omits the categories, so the count comes from the listing.
	wantRenamed := `id          g1
name        Monthly Bills
hidden      false
internal    false
categories  2

renamed category group "Bills" to "Monthly Bills"
`
	if out != wantRenamed {
		t.Errorf("rename =\n%s\nwant\n%s", out, wantRenamed)
	}
	if want := []string{`PATCH /plans/p1/category_groups/g1 {"category_group":{"name":"Monthly Bills"}}`}; !reflect.DeepEqual(h.writes, want) {
		t.Errorf("writes = %v, want %v", h.writes, want)
	}

	refused := []struct {
		args     []string
		wantErr  string
		wantExit int
	}{
		{[]string{"create", "bills"}, `a category group named "Bills" already exists (g1)`, ExitFailure},
		{[]string{"rename", "Credit Card Payments", "Cards"}, "belongs to the API", ExitFailure},
		{[]string{"rename", "Internal Master Category", "Master"}, "belongs to the API", ExitFailure},
		{[]string{"rename", "Fun", "Bills"}, `already exists (g1)`, ExitFailure},
		{[]string{"rename", "Chores", "Fun"}, `category group "Chores" not found`, ExitFailure},
		{[]string{"create", ""}, "NAME must not be empty", ExitUsage},
		{[]string{"create", strings.Repeat("x", 51)}, "NAME is 51 characters; the API allows at most 50", ExitUsage},
		{[]string{"rename", "Bills", strings.Repeat("é", 51)}, "NAME is 51 characters", ExitUsage},
	}
	for _, tc := range refused {
		h := newHarness(t)
		out, err := h.execute(t, append([]string{"category-groups", "--allow-writes"}, tc.args...)...)
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) || ExitCode(err) != tc.wantExit || out != "" || len(h.writes) != 0 {
			t.Errorf("category-groups %v = %q, %v (exit %d), writes %v; want exit %d, an error containing %q, and no write", tc.args, out, err, ExitCode(err), h.writes, tc.wantExit, tc.wantErr)
		}
		if tc.wantExit == ExitUsage && len(h.requests) != 0 {
			t.Errorf("category-groups %v made requests before the usage error: %v", tc.args, h.requests)
		}
	}

	h = newHarness(t)
	if _, err := h.execute(t, "category-groups", "create", strings.Repeat("é", 50), "--dry-run"); err != nil {
		t.Errorf("a 50-character accented name = %v, want accepted", err)
	}
}

func TestPayeesCreateAndRename(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "payees", "create", "Corner Store", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}

	want := "id    pnew\nname  Corner Store\n\ncreated payee \"Corner Store\"\n"
	if out != want {
		t.Errorf("create =\n%s\nwant\n%s", out, want)
	}
	if want := []string{`POST /plans/p1/payees {"payee":{"name":"Corner Store"}}`}; !reflect.DeepEqual(h.writes, want) {
		t.Errorf("writes = %v, want %v", h.writes, want)
	}
	if want := []string{"/plans", "/plans/p1/payees", "/plans/p1/payees"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v", h.requests, want)
	}

	h = newHarness(t)
	out, err = h.execute(t, "payees", "rename", "amazon", "Amazon.com", "--allow-writes", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"id":"p3","name":"Amazon.com"}` + "\n"; out != want {
		t.Errorf("rename --jsonl = %q, want %q", out, want)
	}
	if want := []string{`PATCH /plans/p1/payees/p3 {"payee":{"name":"Amazon.com"}}`}; !reflect.DeepEqual(h.writes, want) {
		t.Errorf("writes = %v, want %v", h.writes, want)
	}

	refused := []struct {
		args     []string
		wantErr  string
		wantExit int
	}{
		{[]string{"create", "costco"}, `a payee named "Costco" already exists (p2)`, ExitFailure},
		{[]string{"rename", "Amazon", "Landlord"}, `already exists (p1)`, ExitFailure},
		{[]string{"rename", "Transfer : Visa", "Card"}, "is a transfer payee", ExitFailure},
		{[]string{"rename", "Nobody", "Somebody"}, `payee "Nobody" not found`, ExitFailure},
		{[]string{"create", strings.Repeat("x", 501)}, "NAME is 501 characters; the API allows at most 500", ExitUsage},
		{[]string{"rename", "Amazon", ""}, "NAME must not be empty", ExitUsage},
	}
	for _, tc := range refused {
		h := newHarness(t)
		out, err := h.execute(t, append([]string{"payees", "--allow-writes"}, tc.args...)...)
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) || ExitCode(err) != tc.wantExit || out != "" || len(h.writes) != 0 {
			t.Errorf("payees %v = %q, %v (exit %d), writes %v; want exit %d, an error containing %q, and no write", tc.args, out, err, ExitCode(err), h.writes, tc.wantExit, tc.wantErr)
		}
		if tc.wantExit == ExitUsage && len(h.requests) != 0 {
			t.Errorf("payees %v made requests before the usage error: %v", tc.args, h.requests)
		}
	}
}

func TestAccountsCreate(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "accounts", "create", "Emergency Fund", "--type", "savings", "--balance", "1000.50", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}

	want := `id                 anew
name               Emergency Fund
type               savings
kind               plan
closed             false
note
balance            $1,000.50
cleared            $1,000.50
uncleared          $0.00
transfer payee id  tpnew
direct import
last reconciled

created savings account "Emergency Fund" with $1,000.50
`
	if out != want {
		t.Errorf("create =\n%s\nwant\n%s", out, want)
	}
	if want := []string{`POST /plans/p1/accounts {"account":{"balance":1000500,"name":"Emergency Fund","type":"savings"}}`}; !reflect.DeepEqual(h.writes, want) {
		t.Errorf("writes = %v, want %v", h.writes, want)
	}

	dry, err := h.execute(t, "accounts", "create", "Car Loan", "--type", "otherLiability", "--balance", "-8000", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dry, "kind               tracking\n") || !strings.Contains(dry, "balance            -$8,000.00\n") || !strings.HasSuffix(dry, `dry run, nothing changed: would create otherLiability account "Car Loan" with -$8,000.00`+"\n") {
		t.Errorf("create --dry-run =\n%s\nwant a tracking preview with the balance", dry)
	}

	refused := []struct {
		args     []string
		wantErr  string
		wantExit int
	}{
		{[]string{"visa", "--type", "cash", "--balance", "0"}, `an account named "Visa" already exists (a2)`, ExitFailure},
		{[]string{"House", "--type", "mortgage", "--balance", "0"}, "invalid --type", ExitUsage},
		{[]string{"House", "--type", "cash", "--balance", "1,000"}, "invalid amount", ExitUsage},
		{[]string{"House", "--type", "cash"}, "balance", ExitUsage},
		{[]string{"", "--type", "cash", "--balance", "0"}, "NAME must not be empty", ExitUsage},
	}
	for _, tc := range refused {
		h := newHarness(t)
		out, err := h.execute(t, append([]string{"accounts", "create", "--allow-writes"}, tc.args...)...)
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) || ExitCode(err) != tc.wantExit || out != "" || len(h.writes) != 0 {
			t.Errorf("accounts create %v = %q, %v (exit %d), writes %v; want exit %d, an error containing %q, and no write", tc.args, out, err, ExitCode(err), h.writes, tc.wantExit, tc.wantErr)
		}
		// The balance parses at the plan's precision, so that check alone
		// costs the plans read.
		if tc.wantExit == ExitUsage && tc.wantErr != "invalid amount" && len(h.requests) != 0 {
			t.Errorf("accounts create %v made requests before the usage error: %v", tc.args, h.requests)
		}
	}
}
