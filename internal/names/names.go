// Package names matches user-supplied names against the exact names of
// plan entities. Matching is case-insensitive and never fuzzy: a near miss
// fails, and Closest exists only to make the failure actionable.
package names

import (
	"sort"
	"strings"
)

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

// Closest returns up to limit candidates ranked by edit distance from
// query, ignoring case, with ties in the candidates' original order.
func Closest(query string, candidates []string, limit int) []string {
	type ranked struct {
		distance int
		index    int
	}
	lowered := strings.ToLower(query)
	ranking := make([]ranked, len(candidates))
	for i, candidate := range candidates {
		ranking[i] = ranked{distance: editDistance(lowered, strings.ToLower(candidate)), index: i}
	}
	sort.SliceStable(ranking, func(i, j int) bool { return ranking[i].distance < ranking[j].distance })

	closest := make([]string, 0, min(limit, len(ranking)))
	for _, entry := range ranking[:cap(closest)] {
		closest = append(closest, candidates[entry.index])
	}
	return closest
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
