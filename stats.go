package leaf

import (
	"math"
	"time"
)

type Stats struct {
	LastReviewedAt time.Time
	Difficulty     float64
	Interval       float64
	initial        bool
}

func DefaultStats() *Stats {
	return &Stats{time.Now(), 0.3, 1, true}
}

func (s *Stats) IsReady() bool {
	if s.initial {
		return true
	}

	nextReviewAt := s.LastReviewedAt.Add(time.Duration(24*s.Interval) * time.Hour)
	return nextReviewAt.Before(time.Now())
}

func (s *Stats) PercentOverdue() float64 {
	percentOverdue := time.Now().Sub(s.LastReviewedAt).Hours() / float64(24*s.Interval)
	return math.Min(2, percentOverdue)
}

func (s *Stats) Record(rating float64) float64 {
	s.initial = false
	success := rating >= 0.6
	percentOverdue := float64(1)
	if success {
		percentOverdue = s.PercentOverdue()
	}

	s.Difficulty += percentOverdue / 50 * (8 - 9*rating)
	s.Difficulty = math.Max(0, math.Min(1, s.Difficulty))
	difficultyWeight := 3.5 - 1.7*s.Difficulty

	minInterval := 0.2
	factor := minInterval / math.Pow(difficultyWeight, 2)
	if success {
		factor = minInterval + (difficultyWeight-1)*percentOverdue
	}

	s.LastReviewedAt = time.Now()
	s.Interval = math.Max(minInterval, math.Min(s.Interval*factor, 300))
	return s.Interval
}
