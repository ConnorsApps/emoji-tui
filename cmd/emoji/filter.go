package main

import (
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"
)

// searchSource flattens emojis into one row per searchable string (the name
// plus each keyword) so a single fuzzy.FindFrom pass scores them all.
//
// fuzzy.FindFrom only knows about a flat indexable list of strings, so we
// keep a parallel slice of searchRow that lets us map a match back to the
// emoji it came from and whether the match was on the name vs a keyword.
type searchSource struct {
	rows []searchRow
}

type searchRow struct {
	emojiIdx int    // index into the original []emoji
	text     string // the string actually searched against
}

// String and Len satisfy fuzzy.Source, the interface FindFrom expects.
func (s searchSource) String(i int) string { return s.rows[i].text }
func (s searchSource) Len() int            { return len(s.rows) }

func buildSource(all []emoji) searchSource {
	// Rough capacity guess: name + a few keywords per emoji. Slightly low is
	// fine; append will grow as needed.
	rows := make([]searchRow, 0, len(all)*4)
	for i, e := range all {
		rows = append(rows, searchRow{i, e.Name})
		for _, k := range e.Keywords {
			rows = append(rows, searchRow{i, k})
		}
	}
	return searchSource{rows: rows}
}

// result is a ranked emoji plus the byte offsets in its Name that matched
// the query, so the UI can highlight those runes.
type result struct {
	Emoji     emoji
	NameMatch []int
}

// filter ranks emojis against the query and returns them best-first.
//
// The query is split on whitespace into independent terms with AND
// semantics: an emoji is kept only if every term fuzzy-matches at least
// one of its rows (name or keyword). Each term independently picks the
// best row for that emoji, and per-term scores are summed so emojis that
// match every term strongly rank above those that just barely qualify.
//
// nameBoost is applied to scores from name rows, so a name hit outranks
// a keyword hit at similar raw fuzzy score.
//
// sahilm/fuzzy uses higher = better, so the final sort is descending.
func filter(all []emoji, src searchSource, q string) []result {
	terms := strings.Fields(q)
	if len(terms) == 0 {
		out := make([]result, len(all))
		for i, e := range all {
			out[i] = result{Emoji: e}
		}
		return out
	}

	type agg struct {
		totalScore   int
		termsMatched int
		nameIdxSet   map[int]struct{}
	}
	aggScore := make(map[int]*agg)

	for termIdx, term := range terms {
		matches := fuzzy.FindFrom(term, src)

		// Best score per emoji for this term
		bestScore := make(map[int]int, len(matches))
		bestNameIdx := make(map[int][]int)
		for _, m := range matches {
			row := src.rows[m.Index]
			score := m.Score
			if cur, ok := bestScore[row.emojiIdx]; !ok || score > cur {
				bestScore[row.emojiIdx] = score
			}
		}

		for emojiIdx, score := range bestScore {
			ag, ok := aggScore[emojiIdx]
			if !ok {
				if termIdx != 0 {
					continue
				}
				ag = &agg{nameIdxSet: make(map[int]struct{})}
				aggScore[emojiIdx] = ag
			}
			ag.termsMatched++
			ag.totalScore += score
			for _, i := range bestNameIdx[emojiIdx] {
				ag.nameIdxSet[i] = struct{}{}
			}
		}
	}

	keep := make([]int, 0, len(aggScore))
	for idx, a := range aggScore {
		if a.termsMatched == len(terms) {
			keep = append(keep, idx)
		}
	}

	sort.Slice(keep, func(i, j int) bool {
		return aggScore[keep[i]].totalScore > aggScore[keep[j]].totalScore
	})

	out := make([]result, len(keep))
	for i, idx := range keep {
		a := aggScore[idx]
		var nm []int
		if len(a.nameIdxSet) > 0 {
			nm = make([]int, 0, len(a.nameIdxSet))
			for k := range a.nameIdxSet {
				nm = append(nm, k)
			}
			sort.Ints(nm)
		}
		out[i] = result{Emoji: all[idx], NameMatch: nm}
	}
	return out
}
