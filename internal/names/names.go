// Package names resolves user-supplied operands against the IDs and exact
// names of plan entities. Matching is case-insensitive and never fuzzy: a
// near miss fails, and the closest names appear only in that failure.
package names

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// closestShown bounds the suggestions in a failed lookup.
const closestShown = 5

// Candidate is one entity an operand can name: its ID and every name that
// identifies it, from the bare form to the most qualified, such as
// "Internet" then "Bills: Internet". The last name is the one errors show.
type Candidate struct {
	ID    string
	Names []string
}

// Resolve returns the index of the candidate the query names: the one
// whose ID equals it, else the one with a name equal to it ignoring case.
// The query is never split, so a name containing ": " still matches as a
// whole. noun names the entity kind in errors. A miss lists the closest
// names; several name matches list every match with its ID.
func Resolve(noun, query string, candidates []Candidate) (int, error) {
	for i, candidate := range candidates {
		if candidate.ID == query {
			return i, nil
		}
	}

	var matches []int
	for i, candidate := range candidates {
		if Match(query, candidate.Names) != nil {
			matches = append(matches, i)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		if len(candidates) == 0 {
			return 0, fmt.Errorf("%s %q not found; the plan has nothing to match", noun, query)
		}
		return 0, fmt.Errorf("%s %q not found; closest names: %s", noun, query, quoteAll(closest(query, candidates)))
	default:
		matched := make([]string, len(matches))
		for i, index := range matches {
			matched[i] = strconv.Quote(qualified(candidates[index])) + " (" + candidates[index].ID + ")"
		}
		return 0, fmt.Errorf("%s %q is ambiguous; use an ID or the full name: %s", noun, query, strings.Join(matched, ", "))
	}
}

// Match returns the indexes of candidates equal to query, ignoring case.
func Match(query string, candidates []string) []int {
	var matches []int
	for i, candidate := range candidates {
		if strings.EqualFold(candidate, query) {
			matches = append(matches, i)
		}
	}
	return matches
}

// closest returns up to closestShown qualified names ranked by the
// smallest edit distance between the query and any of the candidate's
// names, ignoring case, with ties in the candidates' original order.
func closest(query string, candidates []Candidate) []string {
	type ranked struct {
		distance int
		index    int
	}
	lowered := strings.ToLower(query)
	ranking := make([]ranked, len(candidates))
	for i, candidate := range candidates {
		best := -1
		for _, name := range candidate.Names {
			distance := editDistance(lowered, strings.ToLower(name))
			if best < 0 || distance < best {
				best = distance
			}
		}
		ranking[i] = ranked{distance: best, index: i}
	}
	sort.SliceStable(ranking, func(i, j int) bool { return ranking[i].distance < ranking[j].distance })

	limit := min(closestShown, len(ranking))
	names := make([]string, limit)
	for i, entry := range ranking[:limit] {
		names[i] = qualified(candidates[entry.index])
	}
	return names
}

// qualified is the candidate's most qualified name, or its ID when it has
// no names.
func qualified(candidate Candidate) string {
	if len(candidate.Names) == 0 {
		return candidate.ID
	}
	return candidate.Names[len(candidate.Names)-1]
}

func quoteAll(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	return strings.Join(quoted, ", ")
}

// editDistance is the Levenshtein distance over runes.
func editDistance(a, b string) int {
	source, target := []rune(a), []rune(b)
	previous := make([]int, len(target)+1)
	current := make([]int, len(target)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(source); i++ {
		current[0] = i
		for j := 1; j <= len(target); j++ {
			cost := 1
			if source[i-1] == target[j-1] {
				cost = 0
			}
			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}
		previous, current = current, previous
	}
	return previous[len(target)]
}
