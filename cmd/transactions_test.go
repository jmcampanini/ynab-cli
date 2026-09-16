package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

func TestTransactionsListRendersSplitsAndTransfers(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "transactions", "list")
	if err != nil {
		t.Fatal(err)
	}

	want := `DATE        ACCOUNT         PAYEE                      CATEGORY                 MEMO             AMOUNT  CLEARED     APPROVED  FLAG
2026-08-01  Chase Checking  Employer                   Inflow: Ready to Assign                $3,000.00  reconciled  yes
2026-08-20  Chase Checking  Landlord                   Rent                     August rent  -$1,500.00  reconciled  yes
2026-09-02  Chase Checking  Costco                     Split                    weekly run     -$100.00  cleared     yes
                                                         Dining Out                             -$60.00
                                                         Internet               router          -$40.00
2026-09-05  Chase Checking  Transfer : Visa                                                     -$97.81  cleared     yes
2026-09-05  Visa            Transfer : Chase Checking                                            $97.81  cleared     yes
2026-09-10  Visa            Amazon                     Dining Out                               -$12.34  uncleared   no        red
2026-09-11  Chase Checking  New Shop                                            new shop?       -$20.00  uncleared   no
2026-09-12  Chase Checking  Unknown Vendor                                                       -$5.00  cleared     yes
2026-09-13  Chase Checking  Costco                     Split                                    -$25.00  cleared     no
                            Amazon                       Dining Out                             -$10.00
                            Unknown Vendor               Internet               cable           -$15.00
9 transactions, total $1,337.66
`
	if out != want {
		t.Errorf("transactions list =\n%s\nwant\n%s", out, want)
	}
	if want := []string{"/plans", "/plans/p1/accounts", "/plans/p1/transactions"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v", h.requests, want)
	}
}

func TestTransactionsListMachineFormats(t *testing.T) {
	h := newHarness(t)

	jsonl, err := h.execute(t, "transactions", "list", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	csv, err := h.execute(t, "transactions", "list", "--csv")
	if err != nil {
		t.Fatal(err)
	}

	wantRows := []string{
		`{"id":"t1","date":"2026-09-02","account":"Chase Checking","account_id":"a1","payee":"Costco","payee_id":"p2","category":"Split","memo":"weekly run","amount":-100.00,"cleared":"cleared","approved":true,"subtransactions":[{"id":"s1","category":"Dining Out","category_id":"c3","amount":-60.00},{"id":"s2","category":"Internet","category_id":"c1","memo":"router","amount":-40.00}]}`,
		`{"id":"t2","date":"2026-09-05","account":"Chase Checking","account_id":"a1","payee":"Transfer : Visa","payee_id":"tp2","amount":-97.81,"cleared":"cleared","approved":true,"transfer_account":"Visa","transfer_account_id":"a2","transfer_transaction_id":"t2b"}`,
		`{"id":"t3","date":"2026-09-10","account":"Visa","account_id":"a2","payee":"Amazon","payee_id":"p3","category":"Dining Out","category_id":"c3","amount":-12.34,"cleared":"uncleared","approved":false,"flag_color":"red","flag_name":"Reimbursable","import_id":"YNAB:-12340:2026-09-10:1","import_payee_name":"AMAZON.COM","import_payee_name_original":"AMAZON.COM*1234"}`,
		`{"id":"t4","date":"2026-09-12","account":"Chase Checking","account_id":"a1","payee":"Unknown Vendor","payee_id":"p4","amount":-5.00,"cleared":"cleared","approved":true}`,
		`{"id":"t8","date":"2026-09-13","account":"Chase Checking","account_id":"a1","payee":"Costco","payee_id":"p2","category":"Split","amount":-25.00,"cleared":"cleared","approved":false,"subtransactions":[{"id":"s3","payee":"Amazon","payee_id":"p3","category":"Dining Out","category_id":"c3","amount":-10.00},{"id":"s4","payee":"Unknown Vendor","payee_id":"p4","category":"Internet","category_id":"c1","memo":"cable","amount":-15.00}]}`,
	}
	for _, row := range wantRows {
		if !strings.Contains(jsonl, row+"\n") {
			t.Errorf("jsonl lacks row\n%s\nin\n%s", row, jsonl)
		}
	}
	if strings.Count(jsonl, "\n") != 9 || strings.Contains(jsonl, "parent_id") {
		t.Errorf("jsonl = %d rows, want 9 without parent_id:\n%s", strings.Count(jsonl, "\n"), jsonl)
	}

	wantCSV := `id,parent_id,date,account,account_id,payee,payee_id,category,category_id,memo,amount,cleared,approved,flag_color,flag_name,transfer_account,transfer_account_id,transfer_transaction_id,matched_transaction_id,import_id,import_payee_name,import_payee_name_original,debt_transaction_type
t7,,2026-08-01,Chase Checking,a1,Employer,p6,Inflow: Ready to Assign,c7,,3000.00,reconciled,true,,,,,,,,,,
t6,,2026-08-20,Chase Checking,a1,Landlord,p1,Rent,c2,August rent,-1500.00,reconciled,true,,,,,,,,,,
t1,,2026-09-02,Chase Checking,a1,Costco,p2,Split,,weekly run,-100.00,cleared,true,,,,,,,,,,
s1,t1,2026-09-02,Chase Checking,a1,,,Dining Out,c3,,-60.00,cleared,true,,,,,,,,,,
s2,t1,2026-09-02,Chase Checking,a1,,,Internet,c1,router,-40.00,cleared,true,,,,,,,,,,
t2,,2026-09-05,Chase Checking,a1,Transfer : Visa,tp2,,,,-97.81,cleared,true,,,Visa,a2,t2b,,,,,
t2b,,2026-09-05,Visa,a2,Transfer : Chase Checking,tp1,,,,97.81,cleared,true,,,Chase Checking,a1,t2,,,,,
t3,,2026-09-10,Visa,a2,Amazon,p3,Dining Out,c3,,-12.34,uncleared,false,red,Reimbursable,,,,,YNAB:-12340:2026-09-10:1,AMAZON.COM,AMAZON.COM*1234,
t5,,2026-09-11,Chase Checking,a1,New Shop,p5,,,new shop?,-20.00,uncleared,false,,,,,,,YNAB:-20000:2026-09-11:1,NEW SHOP,NEW SHOP 42,
t4,,2026-09-12,Chase Checking,a1,Unknown Vendor,p4,,,,-5.00,cleared,true,,,,,,,,,,
t8,,2026-09-13,Chase Checking,a1,Costco,p2,Split,,,-25.00,cleared,false,,,,,,,,,,
s3,t8,2026-09-13,Chase Checking,a1,Amazon,p3,Dining Out,c3,,-10.00,cleared,false,,,,,,,,,,
s4,t8,2026-09-13,Chase Checking,a1,Unknown Vendor,p4,Internet,c1,cable,-15.00,cleared,false,,,,,,,,,,
`
	if csv != wantCSV {
		t.Errorf("csv =\n%s\nwant\n%s", csv, wantCSV)
	}
}

func TestTransactionFiltersChooseTheListing(t *testing.T) {
	cases := []struct {
		name    string
		filters transactionFilters
		want    listing
	}{
		{"plan", transactionFilters{}, listing{kind: listPlan}},
		{"dates", transactionFilters{since: "2026-08-01", until: "2026-08-31"}, listing{kind: listPlan, query: ynab.TransactionFilter{SinceDate: "2026-08-01", UntilDate: "2026-08-31"}}},
		{"account", transactionFilters{accountID: "a1"}, listing{kind: listAccount}},
		{"month", transactionFilters{month: "2026-08-01"}, listing{kind: listMonth}},
		{"account over month", transactionFilters{accountID: "a1", month: "2026-08-01"}, listing{kind: listAccount, query: ynab.TransactionFilter{SinceDate: "2026-08-01", UntilDate: "2026-08-31"}}},
		{"account, month, and narrower dates", transactionFilters{accountID: "a1", month: "2026-08-01", since: "2026-08-10", until: "2026-09-30"}, listing{kind: listAccount, query: ynab.TransactionFilter{SinceDate: "2026-08-10", UntilDate: "2026-08-31"}}},
		{"category stays client-side", transactionFilters{categoryID: "c3", payeeID: "p2"}, listing{kind: listPlan}},
		{"unapproved", transactionFilters{unapproved: true}, listing{kind: listPlan, query: ynab.TransactionFilter{Type: "unapproved"}}},
		{"uncategorized wins", transactionFilters{unapproved: true, uncategorized: true, month: "2026-09-01"}, listing{kind: listMonth, query: ynab.TransactionFilter{Type: "uncategorized"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.filters.listing()

			if got != tc.want {
				t.Errorf("listing() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestTransactionsListFilters(t *testing.T) {
	cases := []struct {
		args     []string
		wantIDs  string
		wantLast string
		wantURI  string
	}{
		{[]string{"--account", "visa", "--unapproved"}, "t3", "1 transaction, total -$12.34", "/plans/p1/accounts/a2/transactions?type=unapproved"},
		{[]string{"--category", "Dining Out"}, "t1 t3 t8", "3 transactions, matching lines total -$82.34", "/plans/p1/transactions"},
		// t8's lines name other payees, so it is kept through its own payee and counted whole.
		{[]string{"--payee", "costco", "--month", "2026-09"}, "t1 t8", "2 transactions, matching lines total -$125.00", "/plans/p1/months/2026-09-01/transactions"},
		{[]string{"--payee", "amazon"}, "t3 t8", "2 transactions, matching lines total -$22.34", "/plans/p1/transactions"},
		{[]string{"--account", "Chase Checking", "--month", "2026-08", "--since", "2026-08-10"}, "t6", "1 transaction, total -$1,500.00", "/plans/p1/accounts/a1/transactions?since_date=2026-08-10&until_date=2026-08-31"},
		{[]string{"--unapproved", "--uncategorized"}, "t5", "1 transaction, total -$20.00", "/plans/p1/transactions?type=uncategorized"},
		{[]string{"--uncategorized"}, "t5 t4", "2 transactions, total -$25.00", "/plans/p1/transactions?type=uncategorized"},
		{[]string{"--flag", "none", "--cleared", "uncleared"}, "t5", "1 transaction, total -$20.00", "/plans/p1/transactions"},
		{[]string{"--flag", "red"}, "t3", "1 transaction, total -$12.34", "/plans/p1/transactions"},
		{[]string{"--memo", "ROUTER", "--min", "-150", "--max", "-50"}, "t1", "1 transaction, total -$100.00", "/plans/p1/transactions"},
		{[]string{"--min", "0"}, "t7 t2b", "2 transactions, total $3,097.81", "/plans/p1/transactions"},
		{[]string{"--until", "yesterday", "--since", "2026-09-10"}, "t3 t5 t4 t8", "4 transactions, total -$62.34", "/plans/p1/transactions?since_date=2026-09-10&until_date=2026-09-14"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			h := newHarness(t)

			out, err := h.execute(t, append([]string{"transactions", "list", "--jsonl"}, tc.args...)...)
			if err != nil {
				t.Fatal(err)
			}
			table, err := h.execute(t, append([]string{"transactions", "list"}, tc.args...)...)
			if err != nil {
				t.Fatal(err)
			}

			var ids []string
			for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
				ids = append(ids, strings.TrimPrefix(strings.SplitN(line, `","`, 2)[0], `{"id":"`))
			}
			if got := strings.Join(ids, " "); got != tc.wantIDs {
				t.Errorf("ids = %q, want %q", got, tc.wantIDs)
			}
			lines := strings.Split(strings.TrimSpace(table), "\n")
			if got := lines[len(lines)-1]; got != tc.wantLast {
				t.Errorf("summary = %q, want %q", got, tc.wantLast)
			}
			if got := h.requests[len(h.requests)/2-1]; got != tc.wantURI {
				t.Errorf("listing request = %q, want %q", got, tc.wantURI)
			}
		})
	}
}

func TestTransactionsListRejectsBadFilterValues(t *testing.T) {
	h := newHarness(t)
	cases := [][]string{
		{"--cleared", "pending"}, {"--flag", "pink"}, {"--min", "1,000"}, {"--max", "$5"}, {"--since", "last week"},
		{"--until", "2026-13-01"}, {"--month", "September"}, {"--since", "2026-09-10", "--until", "2026-09-01"},
		{"--min", "9223372036854776"},
	}

	for _, args := range cases {
		out, err := h.execute(t, append([]string{"transactions", "list"}, args...)...)
		if err == nil || out != "" || ExitCode(err) != ExitUsage {
			t.Errorf("transactions list %v = %q, %v (exit %d), want a usage error", args, out, err, ExitCode(err))
		}
	}
	if len(h.requests) != 0 {
		t.Errorf("usage errors made requests: %v", h.requests)
	}
}

func TestTransactionsListNameMissesExitOne(t *testing.T) {
	h := newHarness(t)

	for _, args := range [][]string{{"--account", "Chase"}, {"--category", "Internet"}, {"--payee", "Costcos"}} {
		_, err := h.execute(t, append([]string{"transactions", "list"}, args...)...)
		if err == nil || ExitCode(err) != ExitFailure {
			t.Errorf("transactions list %v = %v (exit %d), want exit 1", args, err, ExitCode(err))
		}
	}
	if strings.Contains(strings.Join(h.requests, " "), "/transactions") {
		t.Errorf("a failed name lookup still listed transactions: %v", h.requests)
	}
}

func TestTransactionsGetShowsTransferAndLines(t *testing.T) {
	h := newHarness(t)

	transfer, err := h.execute(t, "transactions", "get", "t2")
	if err != nil {
		t.Fatal(err)
	}
	split, err := h.execute(t, "transactions", "get", "t1")
	if err != nil {
		t.Fatal(err)
	}
	imported, err := h.execute(t, "transactions", "get", "t3", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	flagged, err := h.execute(t, "transactions", "get", "t3")
	if err != nil {
		t.Fatal(err)
	}
	_, missing := h.execute(t, "transactions", "get", "nope")

	wantTransfer := `id                     t2
date                   2026-09-05
account                Chase Checking
payee                  Transfer : Visa
category
memo
amount                 -$97.81
cleared                cleared
approved               yes
flag
transfer account       Visa
transfer transaction   t2b
matched transaction
import id
import payee
import payee original
debt type
`
	if transfer != wantTransfer {
		t.Errorf("transactions get t2 =\n%s\nwant\n%s", transfer, wantTransfer)
	}
	wantLines := `
PAYEE  CATEGORY    MEMO     AMOUNT
       Dining Out          -$60.00
       Internet    router  -$40.00
`
	if !strings.HasSuffix(split, wantLines) || !strings.Contains(split, "category               Split\n") {
		t.Errorf("transactions get t1 =\n%s\nwant lines\n%s", split, wantLines)
	}
	if !strings.Contains(imported, `"flag_color":"red","flag_name":"Reimbursable","import_id":"YNAB:-12340:2026-09-10:1"`) {
		t.Errorf("transactions get t3 --jsonl = %s", imported)
	}
	if !strings.Contains(flagged, "flag                   red (Reimbursable)\n") {
		t.Errorf("transactions get t3 = %s, want the flag color and name", flagged)
	}
	if missing == nil || ExitCode(missing) != ExitFailure || !strings.Contains(missing.Error(), "not found") {
		t.Errorf("transactions get nope = %v (exit %d)", missing, ExitCode(missing))
	}
	if want := []string{"/plans", "/plans/p1/accounts", "/plans/p1/transactions/t2"}; !reflect.DeepEqual(h.requests[:3], want) {
		t.Errorf("requests = %v, want %v", h.requests[:3], want)
	}
}

func TestTransactionsReviewMergesTheTwoListings(t *testing.T) {
	h := newHarness(t)

	table, err := h.execute(t, "transactions", "review")
	if err != nil {
		t.Fatal(err)
	}
	jsonl, err := h.execute(t, "transactions", "review", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	csv, err := h.execute(t, "transactions", "review", "--csv")
	if err != nil {
		t.Fatal(err)
	}

	// Review excludes transfers that need no category even when the API returns them.
	wantTable := `DATE        ACCOUNT         PAYEE           CATEGORY      MEMO        AMOUNT  NEEDS
2026-09-10  Visa            Amazon          Dining Out               -$12.34  approve
2026-09-11  Chase Checking  New Shop                      new shop?  -$20.00  both
2026-09-12  Chase Checking  Unknown Vendor                            -$5.00  categorize
2026-09-13  Chase Checking  Costco          Split                    -$25.00  approve
                            Amazon            Dining Out             -$10.00
                            Unknown Vendor    Internet    cable      -$15.00
4 transactions need attention: 3 to approve, 2 to categorize
`
	if table != wantTable {
		t.Errorf("transactions review =\n%s\nwant\n%s", table, wantTable)
	}
	wantJSONL := `{"id":"t3","date":"2026-09-10","account":"Visa","account_id":"a2","payee":"Amazon","payee_id":"p3","category":"Dining Out","category_id":"c3","amount":-12.34,"cleared":"uncleared","approved":false,"flag_color":"red","flag_name":"Reimbursable","import_id":"YNAB:-12340:2026-09-10:1","import_payee_name":"AMAZON.COM","import_payee_name_original":"AMAZON.COM*1234","needs":"approve"}
{"id":"t5","date":"2026-09-11","account":"Chase Checking","account_id":"a1","payee":"New Shop","payee_id":"p5","memo":"new shop?","amount":-20.00,"cleared":"uncleared","approved":false,"import_id":"YNAB:-20000:2026-09-11:1","import_payee_name":"NEW SHOP","import_payee_name_original":"NEW SHOP 42","needs":"both"}
{"id":"t4","date":"2026-09-12","account":"Chase Checking","account_id":"a1","payee":"Unknown Vendor","payee_id":"p4","amount":-5.00,"cleared":"cleared","approved":true,"needs":"categorize"}
{"id":"t8","date":"2026-09-13","account":"Chase Checking","account_id":"a1","payee":"Costco","payee_id":"p2","category":"Split","amount":-25.00,"cleared":"cleared","approved":false,"subtransactions":[{"id":"s3","payee":"Amazon","payee_id":"p3","category":"Dining Out","category_id":"c3","amount":-10.00},{"id":"s4","payee":"Unknown Vendor","payee_id":"p4","category":"Internet","category_id":"c1","memo":"cable","amount":-15.00}],"needs":"approve"}
`
	if jsonl != wantJSONL {
		t.Errorf("review jsonl =\n%s\nwant\n%s", jsonl, wantJSONL)
	}
	wantCSVTail := `t8,,2026-09-13,Chase Checking,a1,Costco,p2,Split,,,-25.00,cleared,false,,,,,,,,,,,approve
s3,t8,2026-09-13,Chase Checking,a1,Amazon,p3,Dining Out,c3,,-10.00,cleared,false,,,,,,,,,,,approve
s4,t8,2026-09-13,Chase Checking,a1,Unknown Vendor,p4,Internet,c1,cable,-15.00,cleared,false,,,,,,,,,,,approve
`
	if !strings.Contains(csv, ",debt_transaction_type,needs\nt3,,") || !strings.HasSuffix(csv, wantCSVTail) {
		t.Errorf("review csv = %s\nwant header ending in needs and split lines repeating it:\n%s", csv, wantCSVTail)
	}
	want := []string{"/plans", "/plans/p1/accounts", "/plans/p1/transactions?type=unapproved", "/plans/p1/transactions?type=uncategorized"}
	if !reflect.DeepEqual(h.requests[:4], want) {
		t.Errorf("requests = %v, want %v", h.requests[:4], want)
	}
}

func TestReviewSummaryCounts(t *testing.T) {
	if got := reviewSummary(nil); got != "nothing needs attention" {
		t.Errorf("reviewSummary(nil) = %q", got)
	}
	one := []reviewRecord{{Needs: needsBoth}}
	if got := reviewSummary(one); got != "1 transaction needs attention: 1 to approve, 1 to categorize" {
		t.Errorf("reviewSummary(one) = %q", got)
	}
}

func TestReviewCategoryNeedsRespectAccountTypes(t *testing.T) {
	accounts := []ynab.Account{
		{ID: "checking", OnPlan: true},
		{ID: "card", OnPlan: true},
		{ID: "tracking"},
	}
	card, checking, tracking := "card", "checking", "tracking"
	candidates := []ynab.Transaction{
		{ID: "purchase", AccountID: checking},
		{ID: "payment", AccountID: checking, TransferAccountID: &card},
		{ID: "receipt", AccountID: card, TransferAccountID: &checking},
		{ID: "investment", AccountID: checking, TransferAccountID: &tracking},
		{ID: "tracking-receipt", AccountID: tracking, TransferAccountID: &checking},
		{ID: "tracking-only", AccountID: tracking},
	}

	needed := transactionsNeedingCategory(candidates, accounts)
	review := reviewList([]ynab.Transaction{candidates[1]}, needed, newAccountNames(accounts))

	if want := []ynab.Transaction{candidates[0], candidates[3]}; !reflect.DeepEqual(needed, want) {
		t.Errorf("transactionsNeedingCategory() = %+v, want %+v", needed, want)
	}
	if len(review) != 3 || review[0].ID != "payment" || review[0].Needs != needsApprove {
		t.Errorf("reviewList() = %+v, want the unapproved payment plus two category-needed rows", review)
	}
}
