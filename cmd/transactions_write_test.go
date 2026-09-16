package cmd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestWritesExitThreeWithoutAllowWrites(t *testing.T) {
	cases := [][]string{
		{"transactions", "create", "--account", "Visa", "--date", "today", "--amount", "-1"},
		{"transactions", "update", "t3", "--memo", "x"},
		{"transactions", "delete", "t3"},
		{"transactions", "approve", "t3"},
		{"transactions", "approve", "--all-unapproved"},
		{"transactions", "categorize", "--category", "Rent", "t5"},
		{"transactions", "clear", "t3"},
		{"transactions", "unclear", "t3"},
		{"transactions", "flag", "--color", "red", "t3"},
		{"transactions", "import"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args[1:], " "), func(t *testing.T) {
			h := newHarness(t)

			out, err := h.execute(t, args...)

			if err == nil || out != "" || ExitCode(err) != ExitWritesDisabled {
				t.Fatalf("execute(%v) = %q, %v (exit %d), want exit 3 and empty stdout", args, out, err, ExitCode(err))
			}
			for _, want := range []string{"allow_writes", "YNAB_ALLOW_WRITES", "--allow-writes", "ynab.toml"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q lacks %q", err, want)
				}
			}
			if len(h.requests) != 0 {
				t.Errorf("requests = %v, want none", h.requests)
			}
		})
	}
}

func TestAllowWritesFromEnvironmentAndFlagSendTheUpdate(t *testing.T) {
	want := `PATCH /plans/p1/transactions {"transactions":[{"approved":true,"id":"t3"}]}`

	h := newHarness(t)
	t.Setenv("YNAB_ALLOW_WRITES", "true")
	out, err := h.execute(t, "transactions", "approve", "t3")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(h.writes, []string{want}) {
		t.Errorf("writes with YNAB_ALLOW_WRITES = %v, want [%s]", h.writes, want)
	}
	wantOut := `DATE        ACCOUNT  PAYEE   CATEGORY    MEMO   AMOUNT  CLEARED    APPROVED  FLAG
2026-09-10  Visa     Amazon  Dining Out        -$12.34  uncleared  yes       red
approved 1 transaction
`
	if out != wantOut {
		t.Errorf("approve =\n%s\nwant\n%s", out, wantOut)
	}
	if want := []string{"/plans", "/plans/p1/accounts", "/plans/p1/transactions"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v (no read before the update)", h.requests, want)
	}

	h = newHarness(t)
	if _, err := h.execute(t, "transactions", "approve", "t3", "--allow-writes"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(h.writes, []string{want}) {
		t.Errorf("writes with --allow-writes = %v, want [%s]", h.writes, want)
	}
}

func TestDryRunReadsAndPreviewsWithoutWriting(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "transactions", "approve", "t3", "t9", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}

	want := `DATE        ACCOUNT         PAYEE     CATEGORY    MEMO          AMOUNT  CLEARED     APPROVED  FLAG
2026-09-10  Visa            Amazon    Dining Out               -$12.34  uncleared   yes       red
2024-03-02  Chase Checking  Landlord  Rent        old rent  -$1,500.00  reconciled  yes
dry run, nothing changed: would approve 2 transactions
`
	if out != want {
		t.Errorf("approve --dry-run =\n%s\nwant\n%s", out, want)
	}
	// t9 lies outside the listing's default window, so it costs one more read.
	if want := []string{"/plans", "/plans/p1/accounts", "/plans/p1/transactions", "/plans/p1/transactions/t9"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v", h.requests, want)
	}
	if len(h.writes) != 0 {
		t.Errorf("dry run wrote: %v", h.writes)
	}

	jsonl, err := h.execute(t, "transactions", "approve", "t3", "--dry-run", "--allow-writes", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(jsonl, `,"dry_run":true}`+"\n") || !strings.Contains(jsonl, `"approved":true`) || len(h.writes) != 0 {
		t.Errorf("approve --dry-run --jsonl = %q, writes %v; want dry_run marked and nothing written", jsonl, h.writes)
	}

	csv, err := h.execute(t, "transactions", "update", "t1", "--memo", "", "--dry-run", "--csv")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(csv, "id,parent_id,") || !strings.HasSuffix(strings.Split(csv, "\n")[0], ",dry_run") || strings.Count(csv, ",true\n") != 3 {
		t.Errorf("update --dry-run --csv =\n%s\nwant a dry_run column true on the parent and both lines", csv)
	}
}

func TestCreateSendsSingleSplitAndTransferBodies(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantBody string
	}{
		{
			"single with a new payee",
			[]string{"--account", "chase checking", "--date", "yesterday", "--amount", "-97.81", "--payee", "Corner Store", "--category", "Fun: Dining Out", "--memo", "snacks", "--cleared", "cleared", "--approved", "--flag", "blue", "--import-id", "cli:1"},
			`{"account_id":"a1","approved":true,"category_id":"c3","cleared":"cleared","date":"2026-09-14","flag_color":"blue","import_id":"cli:1","memo":"snacks","amount":-97810,"payee_name":"Corner Store"}`,
		},
		{
			"existing payee by name and id",
			[]string{"--account", "a2", "--date", "2026-09-01", "--amount", "+12.5", "--payee", "costco"},
			`{"account_id":"a2","date":"2026-09-01","amount":12500,"payee_id":"p2"}`,
		},
		{
			"split with Group: Name and a memo",
			[]string{"--account", "Visa", "--date", "today", "--amount", "-100", "--payee", "p2", "--split", "Bills: Internet=-40:router", "--split", "Dining Out=-60"},
			`{"account_id":"a2","date":"2026-09-15","amount":-100000,"payee_id":"p2","subtransactions":[{"category_id":"c1","memo":"router","amount":-40000},{"category_id":"c3","amount":-60000}]}`,
		},
		{
			"transfer",
			[]string{"--account", "Chase Checking", "--date", "today", "--amount", "-500", "--transfer-to", "Visa"},
			`{"account_id":"a1","date":"2026-09-15","amount":-500000,"payee_id":"tp2"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)

			out, err := h.execute(t, append([]string{"transactions", "create", "--allow-writes", "--jsonl"}, tc.args...)...)
			if err != nil {
				t.Fatal(err)
			}

			want := `POST /plans/p1/transactions {"transaction":` + tc.wantBody + `}`
			if !reflect.DeepEqual(h.writes, []string{want}) {
				t.Errorf("writes = %v\nwant %s", h.writes, want)
			}
			if !strings.HasPrefix(out, `{"id":"n1",`) || strings.Contains(out, "dry_run") {
				t.Errorf("create --jsonl = %q, want the stored record without a dry_run mark", out)
			}
		})
	}
}

func TestCreateDryRunPrintsThePreview(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "transactions", "create", "--account", "Chase Checking", "--date", "today", "--amount", "-100",
		"--payee", "CLI test", "--split", "Bills: Internet=-40:router", "--split", "Dining Out=-60", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}

	want := `id
date                   2026-09-15
account                Chase Checking
payee                  CLI test
category               Split
memo
amount                 -$100.00
cleared                uncleared
approved               no
flag
transfer account
transfer transaction
matched transaction
import id
import payee
import payee original
debt type

PAYEE  CATEGORY    MEMO     AMOUNT
       Internet    router  -$40.00
       Dining Out          -$60.00

dry run, nothing changed: would create 1 transaction and the payee "CLI test"
`
	if out != want {
		t.Errorf("create --dry-run =\n%s\nwant\n%s", out, want)
	}
	if want := []string{"/plans", "/plans/p1/accounts", "/plans/p1/payees", "/plans/p1/categories"}; !reflect.DeepEqual(h.requests, want) || len(h.writes) != 0 {
		t.Errorf("requests = %v, writes = %v; want %v and no writes", h.requests, h.writes, want)
	}
}

func TestBulkFieldCommandsSendTheirBodies(t *testing.T) {
	cases := []struct {
		args      []string
		wantWrite string
		wantLast  string
	}{
		{[]string{"update", "t3", "t5", "--memo", "", "--flag", "none", "--transfer-to", "Chase Checking"}, `{"transactions":[{"flag_color":null,"id":"t3","memo":"","payee_id":"tp1"},{"flag_color":null,"id":"t5","memo":"","payee_id":"tp1"}]}`, "updated 2 transactions"},
		{[]string{"update", "t3", "--payee", "Corner Store"}, `{"transactions":[{"id":"t3","payee_name":"Corner Store"}]}`, `updated 1 transaction and created the payee "Corner Store"`},
		{[]string{"categorize", "--category", "Bills: Internet", "t5"}, `{"transactions":[{"category_id":"c1","id":"t5"}]}`, "categorized 1 transaction"},
		{[]string{"categorize", "--all-uncategorized", "--category", "c3"}, `{"transactions":[{"category_id":"c3","id":"t5"},{"category_id":"c3","id":"t4"}]}`, "categorized 2 transactions"},
		{[]string{"approve", "--all-unapproved", "--account", "Visa"}, `{"transactions":[{"approved":true,"id":"t3"}]}`, "approved 1 transaction"},
		{[]string{"clear", "t3", "t3"}, `{"transactions":[{"cleared":"cleared","id":"t3"}]}`, "cleared 1 transaction"},
		{[]string{"unclear", "t9", "--force"}, `{"transactions":[{"cleared":"uncleared","id":"t9"}]}`, "uncleared 1 transaction"},
		{[]string{"flag", "--color", "red", "t5"}, `{"transactions":[{"flag_color":"red","id":"t5"}]}`, "flagged 1 transaction"},
		{[]string{"flag", "--color", "none", "t3"}, `{"transactions":[{"flag_color":null,"id":"t3"}]}`, "unflagged 1 transaction"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			h := newHarness(t)

			out, err := h.execute(t, append([]string{"transactions", "--allow-writes"}, tc.args...)...)
			if err != nil {
				t.Fatal(err)
			}

			if want := "PATCH /plans/p1/transactions " + tc.wantWrite; !reflect.DeepEqual(h.writes, []string{want}) {
				t.Errorf("writes = %v\nwant %s", h.writes, want)
			}
			lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
			if lines[len(lines)-1] != tc.wantLast {
				t.Errorf("last line = %q, want %q", lines[len(lines)-1], tc.wantLast)
			}
		})
	}
}

func TestAllUncategorizedSkipsTransfersBetweenPlanAccounts(t *testing.T) {
	h := newHarness(t)

	out, stderr, err := h.executeStreams(t, "transactions", "categorize", "--all-uncategorized", "--category", "Rent", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}

	if stderr != "skipped 2 transactions: these transactions do not need a category\n" {
		t.Errorf("stderr = %q", stderr)
	}
	if strings.Contains(out, "Transfer") || !strings.HasSuffix(out, "would categorize 2 transactions\n") {
		t.Errorf("categorize --all-uncategorized --dry-run =\n%s\nwant t5 and t4 only", out)
	}
}

func TestAllUnapprovedWithNothingToApproveWritesNothing(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "transactions", "approve", "--all-unapproved", "--account", "Old Savings", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(out, "\nnothing to approve\n") || len(h.writes) != 0 {
		t.Errorf("approve with nothing unapproved = %q, writes %v", out, h.writes)
	}
}

func TestClearAndUnclearRefuseReconciledWithoutForce(t *testing.T) {
	for _, verb := range []string{"clear", "unclear"} {
		h := newHarness(t)

		out, err := h.execute(t, "transactions", verb, "t3", "t7", "--allow-writes")

		if err == nil || out != "" || ExitCode(err) != ExitFailure || !strings.Contains(err.Error(), "t7 is reconciled") || !strings.Contains(err.Error(), "--force") {
			t.Errorf("%s of a reconciled transaction = %q, %v (exit %d)", verb, out, err, ExitCode(err))
		}
		if len(h.writes) != 0 {
			t.Errorf("%s wrote before the check: %v", verb, h.writes)
		}
	}
}

func TestUpdateRefusesCategoryOnTransferBetweenPlanAccounts(t *testing.T) {
	h := newHarness(t)

	_, err := h.execute(t, "transactions", "update", "t3", "--transfer-to", "Chase Checking", "--category", "Rent", "--allow-writes")

	if err == nil || ExitCode(err) != ExitUsage || !strings.Contains(err.Error(), "--category is not allowed") || len(h.writes) != 0 {
		t.Errorf("update = %v (exit %d), writes %v; want a usage error before writing", err, ExitCode(err), h.writes)
	}
}

func TestDeletePrintsTheRecordAndImportCounts(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "transactions", "delete", "t3", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"DELETE /plans/p1/transactions/t3 "}; !reflect.DeepEqual(h.writes, want) {
		t.Errorf("writes = %v, want %v", h.writes, want)
	}
	if !strings.HasPrefix(out, "id                     t3\n") || !strings.HasSuffix(out, "\ndeleted 1 transaction\n") {
		t.Errorf("delete =\n%s\nwant the field listing and the summary", out)
	}

	h = newHarness(t)
	out, err = h.execute(t, "transactions", "delete", "t3", "--dry-run", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, `{"id":"t3",`) || !strings.HasSuffix(out, `"dry_run":true}`+"\n") || len(h.writes) != 0 {
		t.Errorf("delete --dry-run --jsonl = %q, writes %v", out, h.writes)
	}
	if want := []string{"/plans", "/plans/p1/accounts", "/plans/p1/transactions/t3"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v", h.requests, want)
	}

	h = newHarness(t)
	out, err = h.execute(t, "transactions", "import", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}
	if out != "imported 2 transactions\n" || !reflect.DeepEqual(h.writes, []string{"POST /plans/p1/transactions/import "}) {
		t.Errorf("import = %q, writes %v", out, h.writes)
	}
	out, err = h.execute(t, "transactions", "import", "--allow-writes", "--jsonl")
	if err != nil || out != `{"count":2,"transaction_ids":["i1","i2"]}`+"\n" {
		t.Errorf("import --jsonl = %q, %v", out, err)
	}
	out, err = h.execute(t, "transactions", "import", "--dry-run")
	if err != nil || out != "dry run, nothing changed: would import from the linked accounts\n" {
		t.Errorf("import --dry-run = %q, %v", out, err)
	}
}

func TestBulkUpdatesBatchAndReportPartialFailure(t *testing.T) {
	ids := make([]string, 250)
	for i := range ids {
		ids[i] = fmt.Sprintf("x%d", i+1)
	}

	h := newHarness(t)
	out, stderr, err := h.executeStreams(t, append([]string{"transactions", "flag", "--color", "green", "--allow-writes", "--jsonl"}, ids...)...)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.writes) != 3 || strings.Count(h.writes[0], `"id"`) != 100 || strings.Count(h.writes[2], `"id"`) != 50 {
		t.Errorf("writes = %d batches of sizes %v, want 100, 100, 50", len(h.writes), idCounts(h.writes))
	}
	if stderr != "batch 1 of 3: 100 transactions\nbatch 2 of 3: 100 transactions\nbatch 3 of 3: 50 transactions\n" {
		t.Errorf("stderr = %q", stderr)
	}
	if strings.Count(out, "\n") != 250 {
		t.Errorf("stdout has %d records, want 250", strings.Count(out, "\n"))
	}

	h = newHarness(t)
	ids[150] = "boom"
	out, _, err = h.executeStreams(t, append([]string{"transactions", "flag", "--color", "green", "--allow-writes"}, ids...)...)
	if err == nil || ExitCode(err) != ExitFailure || !strings.Contains(err.Error(), "batch 2 of 3 failed after 100 transactions were applied") || !strings.Contains(err.Error(), "150 transactions not changed: x101 x102") || !strings.HasSuffix(err.Error(), " x250") {
		t.Errorf("failing batch = %v (exit %d), want exit 1 naming the batch", err, ExitCode(err))
	}
	if len(h.writes) != 2 || strings.Count(out, "\n") != 102 || !strings.HasSuffix(out, "\nflagged 100 transactions\n") {
		t.Errorf("after the failure: %d writes, stdout %d lines ending %q; want 2 writes and the 100 applied records", len(h.writes), strings.Count(out, "\n"), lastLine(out))
	}

	h = newHarness(t)
	_, stderr, err = h.executeStreams(t, "transactions", "flag", "--color", "green", "--allow-writes", "t3")
	if err != nil || stderr != "" {
		t.Errorf("one batch: stderr = %q, err %v; want no progress", stderr, err)
	}
}

func idCounts(writes []string) []int {
	counts := make([]int, len(writes))
	for i, write := range writes {
		counts[i] = strings.Count(write, `"id"`)
	}
	return counts
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	return lines[len(lines)-1]
}

func TestWriteUsageErrorsExitTwoWithoutWriting(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"create", "--date", "today", "--amount", "1"}, "account"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "-1", "--split", "Rent=-1", "--category", "Rent"}, "--split and --category"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "-1", "--split", "Rent"}, "invalid --split"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "-1", "--split", "Rent=-2"}, "--split lines total -2.00 but --amount is -1.00"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "-1", "--import-id", strings.Repeat("x", 37)}, "at most 36"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "1,000"}, "invalid amount"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "1.234"}, "at most 2 decimal places"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "-1", "--payee", "x", "--transfer-to", "Visa"}, "payee"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "-1", "--transfer-to", "Chase Checking", "--category", "Rent"}, "--category is not allowed"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "-1", "--payee", "Transfer : Chase Checking", "--category", "Rent"}, "--category is not allowed"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "-1", "--payee", ""}, "--payee needs"},
		{[]string{"create", "--account", "Visa", "--date", "next week", "--amount", "-1"}, "invalid date"},
		{[]string{"update", "t3"}, "at least one field flag"},
		{[]string{"update", "t3", "--split", "Rent=-1"}, "unknown flag"},
		{[]string{"update", "t3", "--import-id", "x"}, "unknown flag"},
		{[]string{"approve"}, "--all-unapproved"},
		{[]string{"approve", "t3", "--all-unapproved"}, "not both"},
		{[]string{"approve", "t3", "--account", "Visa"}, "--account"},
		{[]string{"categorize", "t3"}, "category"},
		{[]string{"categorize", "--category", "Rent"}, "--all-uncategorized"},
		{[]string{"clear"}, "requires at least 1 arg"},
		{[]string{"flag", "t3"}, "color"},
		{[]string{"flag", "--color", "pink", "t3"}, "invalid --color"},
		{[]string{"delete"}, "accepts 1 arg"},
		{[]string{"import", "extra"}, "unknown command"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			h := newHarness(t)

			out, err := h.execute(t, append([]string{"transactions", "--allow-writes"}, tc.args...)...)

			if err == nil || out != "" || ExitCode(err) != ExitUsage || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("execute = %q, %v (exit %d), want a usage error mentioning %q", out, err, ExitCode(err), tc.want)
			}
			if len(h.writes) != 0 {
				t.Errorf("usage error wrote: %v", h.writes)
			}
		})
	}
}

func TestWriteLookupFailuresExitOne(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "-1", "--category", "Credit Card Payments: Visa"}, "credit card payment category"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "-1", "--transfer-to", "Old Savings"}, "no transfer payee"},
		{[]string{"create", "--account", "Nope", "--date", "today", "--amount", "-1"}, `account "Nope" not found`},
		{[]string{"approve", "nope", "--dry-run"}, "transaction nope: not found"},
		{[]string{"categorize", "--category", "Internet", "t5"}, "ambiguous"},
		{[]string{"create", "--account", "Visa", "--date", "today", "--amount", "-1", "--payee", "3fa85f64-5717-4562-b3fc-2c963f66afa6"}, "payee ID"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			h := newHarness(t)

			out, err := h.execute(t, append([]string{"transactions", "--allow-writes"}, tc.args...)...)

			if err == nil || out != "" || ExitCode(err) != ExitFailure || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("execute = %q, %v (exit %d), want exit 1 mentioning %q", out, err, ExitCode(err), tc.want)
			}
			if len(h.writes) != 0 {
				t.Errorf("lookup failure wrote: %v", h.writes)
			}
		})
	}
}

func TestAllSelectionsListSinceThePlansFirstMonth(t *testing.T) {
	h := newHarness(t)
	if _, err := h.execute(t, "transactions", "approve", "--all-unapproved", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if want := "/plans/p1/transactions?since_date=2024-01-01&type=unapproved"; !reflect.DeepEqual(h.requests[2:], []string{want}) {
		t.Errorf("approve --all-unapproved requests = %v, want %s", h.requests, want)
	}

	h = newHarness(t)
	if _, err := h.execute(t, "transactions", "categorize", "--all-uncategorized", "--category", "Rent", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if want := "/plans/p1/transactions?since_date=2024-01-01&type=uncategorized"; !reflect.DeepEqual(h.requests[3:], []string{want}) {
		t.Errorf("categorize --all-uncategorized requests = %v, want %s", h.requests, want)
	}
}

func TestSplitParentKeepsDateAmountAndCategory(t *testing.T) {
	h := newHarness(t)

	preview, err := h.execute(t, "transactions", "update", "t1", "--amount", "-100", "--category", "Rent", "--date", "2026-09-02", "--memo", "changed", "--dry-run", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{`"date":"2026-09-02"`, `"category":"Split"`, `"amount":-100.00`, `"memo":"changed"`} {
		if !strings.Contains(preview, want) {
			t.Errorf("split preview lacks %s:\n%s", want, preview)
		}
	}

	out, stderr, err := h.executeStreams(t, "transactions", "categorize", "--category", "Rent", "t1", "t3", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "transaction t1 is a split; the API ignored its category change\n" {
		t.Errorf("stderr = %q, want the split warning for t1 only", stderr)
	}
	if !strings.Contains(out, "Split") || !strings.HasSuffix(out, "categorized 2 transactions\n") {
		t.Errorf("categorize with a split =\n%s", out)
	}
}

func TestSplitDateAndAmountChangesFailInDryAndRealRuns(t *testing.T) {
	for _, mode := range []string{"--dry-run", "--allow-writes"} {
		for _, change := range [][]string{{"--amount", "-999"}, {"--date", "2026-01-01"}} {
			t.Run(mode+" "+change[0], func(t *testing.T) {
				h := newHarness(t)
				args := append([]string{"transactions", "update", "t1", mode}, change...)

				out, err := h.execute(t, args...)

				if out != "" || err == nil || ExitCode(err) != ExitFailure || !strings.Contains(err.Error(), "split") {
					t.Errorf("split update = %q, %v; want no records and a split failure", out, err)
				}
				if mode == "--dry-run" && len(h.writes) != 0 {
					t.Errorf("dry run wrote: %v", h.writes)
				}
			})
		}
	}
}

func TestShortBulkAnswerFailsAndPrintsWhatWasApplied(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "transactions", "flag", "--color", "red", "t3", "drop", "--allow-writes")

	if err == nil || ExitCode(err) != ExitFailure || !strings.Contains(err.Error(), "returned 1 of 2 records; missing: drop") || !strings.Contains(err.Error(), "1 transaction not changed: drop") {
		t.Errorf("flag with a dropped record = %v (exit %d)", err, ExitCode(err))
	}
	if !strings.Contains(out, "t3") && !strings.Contains(out, "Amazon") || !strings.HasSuffix(out, "flagged 1 transaction\n") {
		t.Errorf("stdout after the short answer =\n%s\nwant the applied record and its summary", out)
	}
}

func TestCreateClaimsANewPayeeOnlyFromTheStoredRecord(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "transactions", "create", "--account", "Visa", "--date", "today", "--amount", "-1", "--payee", "Corner Store", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out, `created 1 transaction and the payee "Corner Store"`+"\n") {
		t.Errorf("create with a new payee ends %q", lastLine(out))
	}

	out, err = h.execute(t, "transactions", "create", "--account", "Visa", "--date", "today", "--amount", "-1", "--payee", "Costco", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out, "\ncreated 1 transaction\n") {
		t.Errorf("create with an existing payee ends %q", lastLine(out))
	}
}

func TestDuplicateImportIDNamesTheImportID(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "transactions", "create", "--account", "Visa", "--date", "today", "--amount", "-1", "--import-id", "dup", "--allow-writes")

	if err == nil || out != "" || ExitCode(err) != ExitFailure || !strings.Contains(err.Error(), `import id "dup" is already on a transaction: conflict`) {
		t.Errorf("create with a duplicate import id = %q, %v (exit %d)", out, err, ExitCode(err))
	}
}
