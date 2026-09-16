package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// servePlanWrites answers the month, category, category group, payee,
// and account writes of plan p1 the way the API does: it merges the
// request into the fixture row and fills in what the API derives. It
// reports false for paths it does not serve. monthCategories maps an API
// month to its category rows.
func servePlanWrites(t *testing.T, w http.ResponseWriter, r *http.Request, monthCategories map[string]string) bool {
	t.Helper()
	path := strings.TrimPrefix(r.URL.Path, "/plans/p1/")
	// The body is read back into the request for the register handler,
	// which serves the paths this one declines.
	raw, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(raw))
	var body map[string]map[string]any
	_ = json.Unmarshal(raw, &body)
	reply := func(status int, key string, data any) {
		encoded, err := json.Marshal(map[string]any{"data": map[string]any{key: data, "server_knowledge": 1}})
		if err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(status)
		_, _ = w.Write(encoded)
	}
	monthPath := regexp.MustCompile(`^months/(\d{4}-\d{2}-\d{2})/categories/(\w+)$`)

	switch match := monthPath.FindStringSubmatch(path); {
	case r.Method == http.MethodPatch && match != nil:
		raw, ok := fixtureCategoryIn(t, monthCategories[match[1]], match[2])
		if !ok {
			notFound(w)
			return true
		}
		row := decodeRow(t, raw)
		assigned, _ := body["category"]["budgeted"].(float64)
		delta := assigned - row["budgeted"].(float64)
		row["budgeted"] = assigned
		row["balance"] = row["balance"].(float64) + delta
		if underfunded, ok := row["goal_under_funded"].(float64); ok {
			row["goal_under_funded"] = max(0, underfunded-delta)
		}
		reply(http.StatusOK, "category", row)
	case r.Method == http.MethodPost && path == "categories":
		row := map[string]any{"id": "cnew", "hidden": false, "internal": false, "note": nil, "budgeted": 0, "activity": 0, "balance": 0, "goal_type": nil, "goal_target": 0, "deleted": false}
		reply(http.StatusCreated, "category", fixtureStoredCategory(row, body["category"]))
	case r.Method == http.MethodPatch && strings.HasPrefix(path, "categories/"):
		raw, ok := fixtureCategoryIn(t, fixtureCurrentCategories(t), strings.TrimPrefix(path, "categories/"))
		if !ok {
			notFound(w)
			return true
		}
		reply(http.StatusOK, "category", fixtureStoredCategory(decodeRow(t, raw), body["category"]))
	case r.Method == http.MethodPost && path == "category_groups":
		reply(http.StatusCreated, "category_group", map[string]any{"id": "gnew", "name": body["category_group"]["name"], "hidden": false, "internal": false, "deleted": false, "categories": []any{}})
	case r.Method == http.MethodPatch && strings.HasPrefix(path, "category_groups/"):
		// The update answer carries the group without its categories.
		reply(http.StatusOK, "category_group", map[string]any{"id": strings.TrimPrefix(path, "category_groups/"), "name": body["category_group"]["name"], "hidden": false, "internal": false, "deleted": false})
	case r.Method == http.MethodPost && path == "payees":
		reply(http.StatusCreated, "payee", map[string]any{"id": "pnew", "name": body["payee"]["name"], "transfer_account_id": nil, "deleted": false})
	case r.Method == http.MethodPatch && strings.HasPrefix(path, "payees/"):
		reply(http.StatusOK, "payee", map[string]any{"id": strings.TrimPrefix(path, "payees/"), "name": body["payee"]["name"], "transfer_account_id": nil, "deleted": false})
	case r.Method == http.MethodPost && path == "accounts":
		account := body["account"]
		accountType, _ := account["type"].(string)
		reply(http.StatusCreated, "account", map[string]any{
			"id": "anew", "name": account["name"], "type": accountType, "on_budget": accountType != "otherAsset" && accountType != "otherLiability",
			"closed": false, "note": nil, "balance": account["balance"], "cleared_balance": account["balance"], "uncleared_balance": 0,
			"transfer_payee_id": "tpnew", "direct_import_linked": false, "direct_import_in_error": false, "last_reconciled_at": nil, "deleted": false,
		})
	default:
		return false
	}
	return true
}

func decodeRow(t *testing.T, raw string) map[string]any {
	t.Helper()
	var row map[string]any
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	return row
}

// fixtureStoredCategory merges a category save into a row and applies
// the API's goal rules: null removes the target, an amount alone makes a
// NEED target (MF on the credit card payment category), a date makes it
// TBD, and a frequency makes a repeating NEED target.
func fixtureStoredCategory(row, save map[string]any) map[string]any {
	for key, value := range save {
		switch key {
		case "name", "note", "category_group_id", "goal_needs_whole_amount":
			row[key] = value
		}
	}
	if id, ok := row["category_group_id"].(string); ok {
		row["category_group_name"] = fixtureGroupNames[id]
	}
	if target, present := save["goal_target"]; present && target == nil {
		row["goal_type"], row["goal_target"], row["goal_target_date"], row["goal_cadence"], row["goal_cadence_frequency"] = nil, 0, nil, nil, nil
		return row
	}
	if target, ok := save["goal_target"]; ok {
		row["goal_target"] = target
		if row["goal_type"] == nil {
			row["goal_type"], row["goal_cadence"], row["goal_cadence_frequency"] = "NEED", 1, 1
			if row["id"] == "c6" {
				row["goal_type"] = "MF"
			}
		}
	}
	if date, ok := save["goal_target_date"]; ok {
		row["goal_type"], row["goal_target_date"] = "TB", date
	}
	if frequency, ok := save["goal_frequency"].(string); ok {
		row["goal_type"], row["goal_cadence"], row["goal_cadence_frequency"], row["goal_target_date"] = "NEED", map[string]int{"monthly": 1, "weekly": 2, "yearly": 13}[frequency], 1, nil
	}
	return row
}

var fixtureGroupNames = map[string]string{"g1": "Bills", "g2": "Fun", "g3": "Wishes", "g4": "Credit Card Payments", "g5": "Internal Master Category"}
