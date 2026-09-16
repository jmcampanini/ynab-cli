package cmd

import (
	"reflect"
	"strings"
	"testing"
)

// completions runs Cobra's hidden __complete command and returns the
// names it offered, without the directive line.
func completions(t *testing.T, h *harness, args ...string) []string {
	t.Helper()
	out, err := h.execute(t, append([]string{"__complete"}, args...)...)
	if err != nil {
		t.Fatalf("__complete %v: %v", args, err)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if directive := lines[len(lines)-1]; directive != ":4" {
		t.Fatalf("__complete %v ended with %q, want the no-file directive :4", args, directive)
	}
	if len(lines) == 1 {
		return nil
	}
	return lines[:len(lines)-1]
}

func TestCompletionOffersNamesFromThePlan(t *testing.T) {
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"accounts", "get", ""}, []string{"Chase Checking", "Visa"}},
		{[]string{"categories", "get", "r"}, []string{"Rent"}},
		{[]string{"categories", "update", "--group", ""}, []string{"Bills", "Fun", "Credit Card Payments", "Internal Master Category"}},
		{[]string{"payees", "get", "co"}, []string{"Costco"}},
		{[]string{"payees", "rename", "Costco", ""}, nil},
		{[]string{"months", "move", "--from", "R"}, []string{"Rent", "ready-to-assign"}},
		{[]string{"months", "fund", "Rent", "i"}, []string{"Internet"}},
		{[]string{"transactions", "list", "--payee", "a"}, []string{"Amazon"}},
		{[]string{"transactions", "create", "--transfer-to", "v"}, []string{"Visa"}},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			h := newHarness(t)

			got := completions(t, h, tc.args...)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("completions = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCompletionOffersNothingWhenTheRequestFails(t *testing.T) {
	h := newHarness(t)
	t.Setenv("YNAB_TOKEN", "bad-token")

	got := completions(t, h, "categories", "get", "")

	if got != nil {
		t.Errorf("completions with a rejected token = %q, want none", got)
	}
}
