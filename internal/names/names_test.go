package names

import (
	"reflect"
	"testing"
)

func TestMatchIgnoresCaseOnly(t *testing.T) {
	candidates := []string{"Chase Checking", "chase checking", "Chase Savings"}

	if got := Match("CHASE CHECKING", candidates); !reflect.DeepEqual(got, []int{0, 1}) {
		t.Errorf("Match(exact) = %v, want [0 1]", got)
	}
	if got := Match("Chase", candidates); got != nil {
		t.Errorf("Match(prefix) = %v, want no match", got)
	}
}

func TestClosestRanksByEditDistance(t *testing.T) {
	candidates := []string{"Groceries", "Chase Checking", "Chase Savings", "Cash"}

	got := Closest("chase cheking", candidates, 2)

	if want := []string{"Chase Checking", "Chase Savings"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Closest() = %v, want %v", got, want)
	}
	if got := Closest("x", nil, 3); len(got) != 0 {
		t.Errorf("Closest(no candidates) = %v, want empty", got)
	}
}
