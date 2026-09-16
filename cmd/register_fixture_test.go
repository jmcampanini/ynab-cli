package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fixtureTransactions is plan p1's register, oldest first as the API
// returns it: an August inflow and rent, then September's split at Costco,
// a transfer pair between Chase Checking and Visa, an unapproved flagged
// import, a transaction that is both unapproved and uncategorized, an
// uncategorized one, an unapproved split at Costco whose lines each name
// another payee, and a 2024 rent payment that only a since_date or a
// request by ID reaches, since the fake applies the API's one-year
// default window.
const fixtureTransactions = `[
{"id":"t9","date":"2024-03-02","amount":-1500000,"memo":"old rent","cleared":"reconciled","approved":true,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p1","payee_name":"Landlord","category_id":"c2","category_name":"Rent","transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t7","date":"2026-08-01","amount":3000000,"memo":null,"cleared":"reconciled","approved":true,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p6","payee_name":"Employer","category_id":"c7","category_name":"Inflow: Ready to Assign","transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t6","date":"2026-08-20","amount":-1500000,"memo":"August rent","cleared":"reconciled","approved":true,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p1","payee_name":"Landlord","category_id":"c2","category_name":"Rent","transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t1","date":"2026-09-02","amount":-100000,"memo":"weekly run","cleared":"cleared","approved":true,"flag_color":"","flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p2","payee_name":"Costco","category_id":null,"category_name":"Split","transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[
 {"id":"s1","transaction_id":"t1","amount":-60000,"memo":null,"payee_id":null,"payee_name":null,"category_id":"c3","category_name":"Dining Out","transfer_account_id":null,"transfer_transaction_id":null,"deleted":false},
 {"id":"s2","transaction_id":"t1","amount":-40000,"memo":"router","payee_id":null,"payee_name":null,"category_id":"c1","category_name":"Internet","transfer_account_id":null,"transfer_transaction_id":null,"deleted":false}]},
{"id":"t2","date":"2026-09-05","amount":-97810,"memo":null,"cleared":"cleared","approved":true,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"tp2","payee_name":"Transfer : Visa","category_id":null,"category_name":null,"transfer_account_id":"a2","transfer_transaction_id":"t2b","matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t2b","date":"2026-09-05","amount":97810,"memo":null,"cleared":"cleared","approved":true,"flag_color":null,"flag_name":null,"account_id":"a2","account_name":"Visa","payee_id":"tp1","payee_name":"Transfer : Chase Checking","category_id":null,"category_name":null,"transfer_account_id":"a1","transfer_transaction_id":"t2","matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t3","date":"2026-09-10","amount":-12340,"memo":null,"cleared":"uncleared","approved":false,"flag_color":"red","flag_name":"Reimbursable","account_id":"a2","account_name":"Visa","payee_id":"p3","payee_name":"Amazon","category_id":"c3","category_name":"Dining Out","transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":"YNAB:-12340:2026-09-10:1","import_payee_name":"AMAZON.COM","import_payee_name_original":"AMAZON.COM*1234","debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t5","date":"2026-09-11","amount":-20000,"memo":"new shop?","cleared":"uncleared","approved":false,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p5","payee_name":"New Shop","category_id":null,"category_name":null,"transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":"YNAB:-20000:2026-09-11:1","import_payee_name":"NEW SHOP","import_payee_name_original":"NEW SHOP 42","debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t4","date":"2026-09-12","amount":-5000,"memo":null,"cleared":"cleared","approved":true,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p4","payee_name":"Unknown Vendor","category_id":null,"category_name":null,"transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[]},
{"id":"t8","date":"2026-09-13","amount":-25000,"memo":null,"cleared":"cleared","approved":false,"flag_color":null,"flag_name":null,"account_id":"a1","account_name":"Chase Checking","payee_id":"p2","payee_name":"Costco","category_id":null,"category_name":"Split","transfer_account_id":null,"transfer_transaction_id":null,"matched_transaction_id":null,"import_id":null,"import_payee_name":null,"import_payee_name_original":null,"debt_transaction_type":null,"deleted":false,"subtransactions":[
 {"id":"s3","transaction_id":"t8","amount":-10000,"memo":null,"payee_id":"p3","payee_name":"Amazon","category_id":"c3","category_name":"Dining Out","transfer_account_id":null,"transfer_transaction_id":null,"deleted":false},
 {"id":"s4","transaction_id":"t8","amount":-15000,"memo":"cable","payee_id":"p4","payee_name":"Unknown Vendor","category_id":"c1","category_name":"Internet","transfer_account_id":null,"transfer_transaction_id":null,"deleted":false}]}
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

// fixtureMoneyMovements has a September move between categories, an
// August move from ready to assign, and a move back to ready to assign
// the API recorded without a month or time.
const fixtureMoneyMovements = `[
{"id":"mm0","month":null,"moved_at":null,"note":null,"money_movement_group_id":null,"performed_by_user_id":null,"from_category_id":"c2","to_category_id":null,"amount":5000},
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
// since_date and until_date bound the date, the plan and account listings
// default since_date to a year before the harness clock, type unapproved
// keeps unapproved rows, and type uncategorized keeps rows with no
// category and no lines, transfers included as the real API does. It
// reports false for paths it does not own.
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
	listTransactions := func(defaultWindow bool, keep func(fixtureEntry) bool) {
		since := query.Get("since_date")
		if since == "" && defaultWindow {
			since = fixtureNow.AddDate(-1, 0, 0).Format("2006-01-02")
		}
		var kept []fixtureEntry
		for _, entry := range fixtureEntries(t, fixtureTransactions) {
			f := entry.fields
			uncategorized := f.CategoryID == nil && len(f.Subtransactions) == 0
			if !keep(entry) ||
				f.Date < since ||
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
		listTransactions(true, func(fixtureEntry) bool { return true })
	case strings.HasPrefix(path, "transactions/"):
		writeOne("transaction", fixtureTransactions, strings.TrimPrefix(path, "transactions/"))
	case strings.HasPrefix(path, "accounts/") && strings.HasSuffix(path, "/transactions"):
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "accounts/"), "/transactions")
		listTransactions(true, func(e fixtureEntry) bool { return e.fields.AccountID == accountID })
	case strings.HasPrefix(path, "months/") && strings.HasSuffix(path, "/transactions"):
		month := strings.TrimSuffix(strings.TrimPrefix(path, "months/"), "/transactions")
		listTransactions(false, func(e fixtureEntry) bool { return strings.HasPrefix(e.fields.Date, month[:7]) })
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

// Names the fake fills in on a stored transaction, as the API does.
var (
	fixtureAccountNames  = map[string]string{"a1": "Chase Checking", "a2": "Visa", "a3": "Old Savings"}
	fixtureCategoryNames = map[string]string{"c1": "Internet", "c2": "Rent", "c3": "Dining Out", "c4": "Old Hobby", "c5": "Internet", "c6": "Visa", "c7": "Inflow: Ready to Assign"}
)

// serveRegisterWrites answers the transaction writes of plan p1 the way
// the API does, without keeping state: a create returns the body as
// stored under the id "n1"; a bulk update merges each entry into the
// fixture row its id names, or into a copy of t4 for an id the fixture
// lacks, so a test can send hundreds of ids; a delete returns the row;
// an import returns two ids. An update entry whose id is "boom" fails
// with 500 so a later batch can fail, and one whose id is "drop" is left
// out of the answer. A create whose import_id is "dup" is a 409.
func serveRegisterWrites(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	path := strings.TrimPrefix(r.URL.Path, "/plans/p1/")
	var body struct {
		Transaction  map[string]any   `json:"transaction"`
		Transactions []map[string]any `json:"transactions"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	rows := map[string]map[string]any{}
	for _, entry := range fixtureEntries(t, fixtureTransactions) {
		var row map[string]any
		if err := json.Unmarshal(entry.raw, &row); err != nil {
			t.Fatal(err)
		}
		rows[entry.fields.ID] = row
	}
	reply := func(status int, data any) {
		encoded, err := json.Marshal(map[string]any{"data": data})
		if err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(status)
		_, _ = w.Write(encoded)
	}

	switch {
	case r.Method == http.MethodPost && path == "transactions":
		if body.Transaction["import_id"] == "dup" {
			w.WriteHeader(http.StatusConflict)
			_, _ = io.WriteString(w, `{"error":{"id":"409","name":"conflict","detail":"Conflict"}}`)
			return
		}
		stored := fixtureStored(map[string]any{"id": "n1", "cleared": "uncleared", "approved": false, "subtransactions": []any{}}, body.Transaction)
		reply(http.StatusCreated, map[string]any{"transaction_ids": []string{"n1"}, "transaction": stored, "duplicate_import_ids": []string{}})
	case r.Method == http.MethodPatch && path == "transactions":
		var stored []map[string]any
		for _, entry := range body.Transactions {
			id, _ := entry["id"].(string)
			if id == "boom" {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = io.WriteString(w, `{"error":{"id":"500","name":"internal_server_error","detail":"boom"}}`)
				return
			}
			if id == "drop" {
				continue
			}
			base, ok := rows[id]
			if !ok {
				base = map[string]any{}
				for key, value := range rows["t4"] {
					base[key] = value
				}
				base["id"] = id
			}
			if lines, _ := base["subtransactions"].([]any); len(lines) > 0 {
				for _, field := range []string{"date", "amount"} {
					if value, changed := entry[field]; changed && value != base[field] {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = io.WriteString(w, `{"error":{"id":"400","name":"bad_request","detail":"the amount or date of an existing split transaction cannot be changed"}}`)
						return
					}
				}
			}
			stored = append(stored, fixtureStored(base, entry))
		}
		reply(http.StatusOK, map[string]any{"transaction_ids": []string{}, "transactions": stored, "duplicate_import_ids": []string{}})
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "transactions/"):
		row, ok := rows[strings.TrimPrefix(path, "transactions/")]
		if !ok {
			notFound(w)
			return
		}
		reply(http.StatusOK, map[string]any{"transaction": row})
	case r.Method == http.MethodPost && path == "transactions/import":
		reply(http.StatusCreated, map[string]any{"transaction_ids": []string{"i1", "i2"}})
	default:
		notFound(w)
	}
}

// fixtureStored merges a request entry into a row as the API does: on an
// existing split the category is ignored. The handler rejects changes
// to its date or amount before this merge. It fills in
// the names the API derives: the account, category, and payee names, the
// transfer account behind a transfer payee, a new payee's id, and split
// line ids.
func fixtureStored(row, entry map[string]any) map[string]any {
	lines, _ := row["subtransactions"].([]any)
	for key, value := range entry {
		if len(lines) > 0 && (key == "date" || key == "amount" || key == "category_id") {
			continue
		}
		row[key] = value
	}
	if id, ok := row["account_id"].(string); ok {
		row["account_name"] = fixtureAccountNames[id]
	}
	if id, ok := row["category_id"].(string); ok {
		row["category_name"] = fixtureCategoryNames[id]
	}
	if name, ok := entry["payee_name"].(string); ok {
		row["payee_id"], row["payee_name"] = "pnew", name
	}
	if id, ok := entry["payee_id"].(string); ok {
		if account, isTransfer := strings.CutPrefix(id, "tp"); isTransfer {
			row["payee_name"] = "Transfer : " + fixtureAccountNames["a"+account]
			row["transfer_account_id"] = "a" + account
		}
		for _, payee := range fixturePayeeNames() {
			if payee[0] == id {
				row["payee_name"] = payee[1]
			}
		}
	}
	if lines, ok := entry["subtransactions"].([]any); ok {
		var stored []any
		row["category_id"], row["category_name"] = nil, "Split"
		for i, line := range lines {
			fields, _ := line.(map[string]any)
			fields["id"] = fmt.Sprintf("n1s%d", i+1)
			if id, ok := fields["category_id"].(string); ok {
				fields["category_name"] = fixtureCategoryNames[id]
			}
			stored = append(stored, fields)
		}
		row["subtransactions"] = stored
	}
	return row
}

// fixturePayeeNames lists the payee fixture as id, name pairs.
func fixturePayeeNames() [][2]string {
	var data struct {
		Data struct {
			Payees []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"payees"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(fixturePayees), &data)
	pairs := make([][2]string, len(data.Data.Payees))
	for i, payee := range data.Data.Payees {
		pairs[i] = [2]string{payee.ID, payee.Name}
	}
	return pairs
}
