package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// fixtureTransactions is plan p1's register, oldest first as the API
// returns it: an August inflow and rent, then September's split at Costco,
// a transfer pair between Chase Checking and Visa, an unapproved flagged
// import, a transaction that is both unapproved and uncategorized, and an
// uncategorized one.
const fixtureTransactions = `[
{"id":"t7","date":"2026-08-01","amount":3000000,"memo":null,"cleared":"reconciled","approved":true,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p6","payee_name":"Employer","category_id":"c7","category_name":"Inflow: Ready to Assign","transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t6","date":"2026-08-20","amount":-1500000,"memo":"August rent","cleared":"reconciled","approved":true,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p1","payee_name":"Landlord","category_id":"c2","category_name":"Rent","transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t1","date":"2026-09-02","amount":-100000,"memo":"weekly run","cleared":"cleared","approved":true,"flag_color":"","flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p2","payee_name":"Costco","category_id":null,"category_name":"Split","transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[
 {"id":"s1","transaction_id":"t1","amount":-60000,"memo":null,"payee_id":null,"payee_name":null,"category_id":"c3","category_name":"Dining Out","transfer_account_id":null,"transfer_transaction_id":null,"deleted":false},
 {"id":"s2","transaction_id":"t1","amount":-40000,"memo":"router","payee_id":null,"payee_name":null,"category_id":"c1","category_name":"Internet","transfer_account_id":null,"transfer_transaction_id":null,"deleted":false}]},
{"id":"t2","date":"2026-09-05","amount":-97810,"memo":null,"cleared":"cleared","approved":true,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"tp2","payee_name":"Transfer : Visa","category_id":null,"category_name":null,"transfer_account_id":"a2","transfer_transaction_id":"t2b","matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t2b","date":"2026-09-05","amount":97810,"memo":null,"cleared":"cleared","approved":true,"flag_color":null,"flag_name":null,"account_id":"a2","account_name":"Visa","payee_id":"tp1","payee_name":"Transfer : Chase Checking","category_id":null,"category_name":null,"transfer_account_id":"a1","transfer_transaction_id":"t2","matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t3","date":"2026-09-10","amount":-12340,"memo":null,"cleared":"uncleared","approved":false,"flag_color":"red","flag_name":"Reimbursable","account_id":"a2","account_name":"Visa","payee_id":"p3","payee_name":"Amazon","category_id":"c3","category_name":"Dining Out","transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":"YNAB:-12340:2026-09-10:1","import_payee_name":"AMAZON.COM","import_payee_name_original":"AMAZON.COM*1234","debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t5","date":"2026-09-11","amount":-20000,"memo":"new shop?","cleared":"uncleared","approved":false,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p5","payee_name":"New Shop","category_id":null,"category_name":null,"transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":"YNAB:-20000:2026-09-11:1","import_payee_name":"NEW SHOP","import_payee_name_original":"NEW SHOP 42","debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t4","date":"2026-09-12","amount":-5000,"memo":null,"cleared":"cleared","approved":true,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p4","payee_name":"Unknown Vendor","category_id":null,"category_name":null,"transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[]}
]`

// fixturePayees includes the two transfer payees, one payee used only by
// a scheduled transaction (Insurance Co), and one used nowhere (Old
// Bakery).
const fixturePayees = `{"data":{"payees":[
{"id":"p1","name":"Landlord","transfer_account_id":null,"deleted":false},
{"id":"p2","name":"Costco","transfer_account_id":null,"deleted":false},
{"id":"p3","name":"Amazon","transfer_account_id":null,"deleted":false},
{"id":"p4","name":"Unknown Vendor","transfer_account_id":null,"deleted":false},
{"id":"p5","name":"New Shop","transfer_account_id":null,"deleted":false},
{"id":"p6","name":"Employer","transfer_account_id":null,"deleted":false},
{"id":"p7","name":"Old Bakery","transfer_account_id":null,"deleted":false},
{"id":"p8","name":"Insurance Co","transfer_account_id":null,"deleted":false},
{"id":"tp1","name":"Transfer : Chase Checking","transfer_account_id":"a1","deleted":false},
{"id":"tp2","name":"Transfer : Visa","transfer_account_id":"a2","deleted":false}
]}}`

// fixtureScheduled has a monthly rent and a yearly split.
const fixtureScheduled = `[
{"id":"st1","date_first":"2024-02-01","date_next":"2026-10-01","frequency":"monthly","amount":-1500000,"memo":null,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p1","payee_name":"Landlord","category_id":"c2","category_name":"Rent","transfer_account_id":null,"deleted":false,"subtransactions":[]},
{"id":"st2","date_first":"2027-01-15","date_next":"2027-01-15","frequency":"yearly","amount":-600000,"memo":"annual","flag_color":"blue","flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p8","payee_name":"Insurance Co","category_id":null,"category_name":"Split","transfer_account_id":null,"deleted":false,"subtransactions":[
 {"id":"ss1","scheduled_transaction_id":"st2","amount":-100000,"memo":null,"payee_id":null,"payee_name":null,"category_id":"c3","category_name":"Dining Out","transfer_account_id":null,"deleted":false},
 {"id":"ss2","scheduled_transaction_id":"st2","amount":-500000,"memo":"modem","payee_id":null,"payee_name":null,"category_id":"c1","category_name":"Internet","transfer_account_id":null,"deleted":false}]}
]`

// fixtureMoneyMovements has a September move between categories and an
// August move from ready to assign.
const fixtureMoneyMovements = `[
{"id":"mm2","month":"2026-08-01","moved_at":"2026-08-01T08:00:00Z","note":null,"money_movement_group_id":null,"performed_by_user_id":"u1","from_category_id":null,"to_category_id":"c2","amount":1500000},
{"id":"mm1","month":"2026-09-01","moved_at":"2026-09-03T10:00:00Z","note":"cover dining","money_movement_group_id":"mg1","performed_by_user_id":"u1","from_category_id":"c3","to_category_id":"c1","amount":25000}
]`

// fixtureEntry is one raw fixture row plus the fields the fake filters by.
type fixtureEntry struct {
	raw    json.RawMessage
	fields struct {
		AccountID         string            `json:"account_id"`
		Approved          bool              `json:"approved"`
		CategoryID        *string           `json:"category_id"`
		Date              string            `json:"date"`
		ID                string            `json:"id"`
		Month             string            `json:"month"`
		Subtransactions   []json.RawMessage `json:"subtransactions"`
		TransferAccountID *string           `json:"transfer_account_id"`
	}
}

func fixtureEntries(t *testing.T, array string) []fixtureEntry {
	t.Helper()
	var raws []json.RawMessage
	if err := json.Unmarshal([]byte(array), &raws); err != nil {
		t.Fatal(err)
	}
	entries := make([]fixtureEntry, len(raws))
	for i, raw := range raws {
		entries[i].raw = raw
		if err := json.Unmarshal(raw, &entries[i].fields); err != nil {
			t.Fatal(err)
		}
	}
	return entries
}

func fixtureArray(entries []fixtureEntry) string {
	parts := make([]string, len(entries))
	for i, entry := range entries {
		parts[i] = string(entry.raw)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// serveRegister answers the transaction, payee, scheduled, and money
// movement routes of plan p1, applying the listing query as the API does:
// since_date and until_date bound the date, type unapproved keeps
// unapproved rows, and type uncategorized keeps rows with no category, no
// lines, and no transfer. It reports false for paths it does not own.
func serveRegister(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	t.Helper()
	path := strings.TrimPrefix(r.URL.Path, "/plans/p1/")
	query := r.URL.Query()
	writeList := func(key string, entries []fixtureEntry) {
		_, _ = w.Write([]byte(`{"data":{"` + key + `":` + fixtureArray(entries) + `}}`))
	}
	writeOne := func(key, array, id string) {
		for _, entry := range fixtureEntries(t, array) {
			if entry.fields.ID == id {
				_, _ = w.Write([]byte(`{"data":{"` + key + `":` + string(entry.raw) + `}}`))
				return
			}
		}
		notFound(w)
	}
	listTransactions := func(keep func(fixtureEntry) bool) {
		var kept []fixtureEntry
		for _, entry := range fixtureEntries(t, fixtureTransactions) {
			f := entry.fields
			uncategorized := f.CategoryID == nil && len(f.Subtransactions) == 0 && f.TransferAccountID == nil
			if !keep(entry) ||
				query.Get("since_date") != "" && f.Date < query.Get("since_date") ||
				query.Get("until_date") != "" && f.Date > query.Get("until_date") ||
				query.Get("type") == "unapproved" && f.Approved ||
				query.Get("type") == "uncategorized" && !uncategorized {
				continue
			}
			kept = append(kept, entry)
		}
		writeList("transactions", kept)
	}

	switch {
	case path == "transactions":
		listTransactions(func(fixtureEntry) bool { return true })
	case strings.HasPrefix(path, "transactions/"):
		writeOne("transaction", fixtureTransactions, strings.TrimPrefix(path, "transactions/"))
	case strings.HasPrefix(path, "accounts/") && strings.HasSuffix(path, "/transactions"):
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "accounts/"), "/transactions")
		listTransactions(func(e fixtureEntry) bool { return e.fields.AccountID == accountID })
	case strings.HasPrefix(path, "months/") && strings.HasSuffix(path, "/transactions"):
		month := strings.TrimSuffix(strings.TrimPrefix(path, "months/"), "/transactions")
		listTransactions(func(e fixtureEntry) bool { return strings.HasPrefix(e.fields.Date, month[:7]) })
	case path == "payees":
		_, _ = w.Write([]byte(fixturePayees))
	case path == "scheduled_transactions":
		writeList("scheduled_transactions", fixtureEntries(t, fixtureScheduled))
	case strings.HasPrefix(path, "scheduled_transactions/"):
		writeOne("scheduled_transaction", fixtureScheduled, strings.TrimPrefix(path, "scheduled_transactions/"))
	case path == "money_movements":
		writeList("money_movements", fixtureEntries(t, fixtureMoneyMovements))
	case strings.HasPrefix(path, "months/") && strings.HasSuffix(path, "/money_movements"):
		month := strings.TrimSuffix(strings.TrimPrefix(path, "months/"), "/money_movements")
		var kept []fixtureEntry
		for _, entry := range fixtureEntries(t, fixtureMoneyMovements) {
			if entry.fields.Month == month {
				kept = append(kept, entry)
			}
		}
		writeList("money_movements", kept)
	default:
		return false
	}
	return true
}
