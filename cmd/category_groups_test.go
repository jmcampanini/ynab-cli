package cmd

import (
	"strings"
	"testing"
)

func TestCategoryGroupsListCountsWhatCategoriesListShows(t *testing.T) {
	h := newHarness(t)

	visible, err := h.execute(t, "category-groups", "list")
	if err != nil {
		t.Fatal(err)
	}
	all, err := h.execute(t, "category-groups", "list", "--hidden")
	if err != nil {
		t.Fatal(err)
	}
	jsonl, err := h.execute(t, "category-groups", "list", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	wantVisible := `NAME                      CATEGORIES  ID
Bills                              2  g1
Fun                                1  g2
Credit Card Payments               1  g4
Internal Master Category           1  g5
4 groups, 1 hidden not shown
`
	if visible != wantVisible {
		t.Errorf("category-groups list =\n%s\nwant\n%s", visible, wantVisible)
	}
	wantAll := `NAME                      CATEGORIES  ID
Bills                              2  g1
Fun                                2  g2
Wishes (hidden)                    1  g3
Credit Card Payments               1  g4
Internal Master Category           1  g5
5 groups
`
	if all != wantAll {
		t.Errorf("category-groups list --hidden =\n%s\nwant\n%s", all, wantAll)
	}
	if !strings.HasPrefix(jsonl, `{"id":"g1","name":"Bills","hidden":false,"internal":false,"category_count":2}`+"\n") || strings.Count(jsonl, "\n") != 4 {
		t.Errorf("jsonl =\n%s", jsonl)
	}
}
