package memory

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/laughingmandev/loa/internal/state"
)

type Candidate struct {
	ID        uint64           `json:"id"`
	Kind      state.MemoryKind `json:"kind"`
	Text      string           `json:"text"`
	CreatedAt time.Time        `json:"created_at"`
	Score     float64          `json:"score"`
}

func Search(items []state.MemoryItem, summaries []state.TaskSummary, query []float32, limit int, now time.Time, targets []string, halfLifeHours float64, maxWeight float64, allowedKinds []state.MemoryKind) []Candidate {
	if limit <= 0 {
		return nil
	}
	out := make([]Candidate, 0, len(items)+len(summaries))
	for _, item := range items {

		if len(item.Embedding) == 0 {
			continue
		}
		
		if len(allowedKinds) > 0 {
			allowed := false
			for _, k := range allowedKinds {
				if item.Kind == k {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		// Phase 4.2: Hard Filter for structural memories based on targets
		if item.Kind == state.MemoryStructural && len(targets) > 0 {
			matched := false
			for _, target := range targets {
				for _, anchor := range item.Anchors {
					if strings.Contains(anchor, target) || strings.Contains(target, anchor) {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				continue // Skip this item as it doesn't match any of the target entities
			}
		}

		out = append(out, Candidate{ID: item.ID, Kind: item.Kind, Text: item.Text, CreatedAt: item.CreatedAt, Score: score(query, item.Embedding, item.CreatedAt, now, halfLifeHours, maxWeight)})
	}
	for _, s := range summaries {
		if len(s.Embedding) == 0 {
			continue
		}
		
		if len(allowedKinds) > 0 {
			allowed := false
			for _, k := range allowedKinds {
				if state.MemoryTask == k {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}
		out = append(out, Candidate{ID: s.TaskID, Kind: state.MemoryTask, Text: s.Summary, CreatedAt: s.CreatedAt, Score: score(query, s.Embedding, s.CreatedAt, now, halfLifeHours, maxWeight)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func score(a, b []float32, created, now time.Time, halfLifeHours float64, maxWeight float64) float64 {
	sim := cosine(a, b)
	ageHours := now.Sub(created).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	recency := maxWeight * math.Exp(-ageHours/halfLifeHours)
	return sim + recency
}

func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return -1
	}
	var dot, aa, bb float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		aa += x * x
		bb += y * y
	}
	if aa == 0 || bb == 0 {
		return -1
	}
	return dot / (math.Sqrt(aa) * math.Sqrt(bb))
}
