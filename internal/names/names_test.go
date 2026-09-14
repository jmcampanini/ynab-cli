package names

import (
	"reflect"
	"strings"
	"testing"
)

var categories = []Candidate{
	{ID: "c1", Names: []string{"Internet", "Bills: Internet"}},
	{ID: "c2", Names: []string{"Rent", "Bills: Rent"}},
	{ID: "c3", Names: []string{"Internet", "Wishes: Internet"}},
	{ID: "c4", Names: []string{"Inflow: Ready to Assign", "Internal Master Category: Inflow: Ready to Assign"}},
}

func TestResolveMatchesIDThenExactNameIgnoringCase(t *testing.T) {
	cases := []struct {
		query string
		want  int
	}{
		{"c3", 2},
		{"rent", 1},
		{"BILLS: internet", 0},
		{"inflow: ready to assign", 3},
	}
	for _, tc := range cases {
		got, err := Resolve("category", tc.query, categories)
		if err != nil || got != tc.want {
			t.Errorf("Resolve(%q) = %d, %v; want %d", tc.query, got, err, tc.want)
		}
	}
}

func TestResolveRejectsPrefixAndListsClosestQualifiedNames(t *testing.T) {
	_, err := Resolve("category", "Ren", categories)

	if err == nil {
		t.Fatal("Resolve(prefix) succeeded; want a miss")
	}
	want := `category "Ren" not found; closest names: "Bills: Rent", "Bills: Internet", "Wishes: Internet", "Internal Master Category: Inflow: Ready to Assign"`
	if err.Error() != want {
		t.Errorf("Resolve(prefix) error = %q, want %q", err.Error(), want)
	}
}

func TestResolveAmbiguousNameListsEveryFullName(t *testing.T) {
	_, err := Resolve("category", "internet", categories)

	if err == nil {
		t.Fatal("Resolve(shared name) succeeded; want ambiguity")
	}
	want := `category "internet" is ambiguous; use an ID or the full name: "Bills: Internet" (c1), "Wishes: Internet" (c3)`
	if err.Error() != want {
		t.Errorf("Resolve(shared name) error = %q, want %q", err.Error(), want)
	}
}

func TestResolveWithNoCandidatesSaysSo(t *testing.T) {
	_, err := Resolve("account", "Cash", nil)

	if err == nil || !strings.Contains(err.Error(), "nothing to match") {
		t.Errorf("Resolve(no candidates) error = %v", err)
	}
}

func TestClosestCapsAtFiveByEditDistance(t *testing.T) {
	var many []Candidate
	for _, name := range []string{"Groceries", "Chase Checking", "Chase Savings", "Cash", "Car", "Camp", "Zoo"} {
		many = append(many, Candidate{ID: name, Names: []string{name}})
	}

	got := closest("chase cheking", many)

	if len(got) != 5 || got[0] != "Chase Checking" || got[1] != "Chase Savings" {
		t.Errorf("closest() = %v, want five names led by Chase Checking then Chase Savings", got)
	}
}

func TestMatchIgnoresCaseOnly(t *testing.T) {
	candidates := []string{"Chase Checking", "chase checking", "Chase Savings"}

	if got := Match("CHASE CHECKING", candidates); !reflect.DeepEqual(got, []int{0, 1}) {
		t.Errorf("Match(exact) = %v, want [0 1]", got)
	}
	if got := Match("Chase", candidates); got != nil {
		t.Errorf("Match(prefix) = %v, want no match", got)
	}
}
