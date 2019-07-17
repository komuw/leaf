package leaf

import (
	"encoding/json"
	"math"
	"time"
)

// Supermemo calculates review intervals
type Supermemo interface {
	json.Marshaler
	json.Unmarshaler

	// Advance advances supermemo state for a card.
	Advance(rating float64) (interval float64)
	// NextReviewAt returns next review timestamp for a card.
	NextReviewAt() time.Time
	// SortParam returns values that should used as a review order for cards
	SortParam() float64
}

// IntervalSnapshot records historical changes of the Interval.
type IntervalSnapshot struct {
	Timestamp  int64   `json:"ts"`
	Interval   float64 `json:"interval"`
	Difficulty float64 `json:"difficulty"`
}

// Supermemo2Plus calculates review intervals using SM2+ algorithm
type Supermemo2Plus struct {
	LastReviewedAt time.Time
	Difficulty     float64
	Interval       float64
	Historical     []IntervalSnapshot
}

// NewSupermemo2Plus returns a new Supermemo2Plus instance
func NewSupermemo2Plus() *Supermemo2Plus {
	return &Supermemo2Plus{
		LastReviewedAt: time.Now().Add(-5 * time.Hour),
		Difficulty:     0.3,
		Interval:       0.2,
		Historical:     make([]IntervalSnapshot, 0),
	}
}

// NextReviewAt returns next review timestamp for a card.
func (sm *Supermemo2Plus) NextReviewAt() time.Time {
	return sm.LastReviewedAt.Add(time.Duration(24*sm.Interval) * time.Hour)
}

// SortParam returns values that should used as a review order for cards
func (sm *Supermemo2Plus) SortParam() float64 {
	return sm.PercentOverdue()
}

// PercentOverdue returns corresponding SM2+ value for a Card.
func (sm *Supermemo2Plus) PercentOverdue() float64 {
	percentOverdue := time.Since(sm.LastReviewedAt).Hours() / float64(24*sm.Interval)
	return math.Min(2, percentOverdue)
}

// Advance advances supermemo state for a card.
func (sm *Supermemo2Plus) Advance(rating float64) float64 {
	success := rating >= ratingSuccess
	percentOverdue := float64(1)
	if success {
		percentOverdue = sm.PercentOverdue()
	}

	sm.Difficulty += percentOverdue / 50 * (8 - 9*rating)
	sm.Difficulty = math.Max(0, math.Min(1, sm.Difficulty))
	difficultyWeight := 3.5 - 1.7*sm.Difficulty

	minInterval := math.Min(1.0, sm.Interval)
	factor := minInterval / math.Pow(difficultyWeight, 2)
	if success {
		minInterval = 0.2
		factor = minInterval + (difficultyWeight-1)*percentOverdue
	}

	sm.LastReviewedAt = time.Now()
	if sm.Historical == nil {
		sm.Historical = make([]IntervalSnapshot, 0)
	}
	sm.Historical = append(
		sm.Historical,
		IntervalSnapshot{time.Now().Unix(), sm.Interval, sm.Difficulty},
	)
	sm.Interval = math.Max(minInterval, math.Min(sm.Interval*factor, 300))
	return sm.Interval
}

// MarshalJSON implements json.Marshaller for Supermemo2Plus
func (sm *Supermemo2Plus) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		LastReviewedAt time.Time
		Difficulty     float64
		Interval       float64
		Historical     []IntervalSnapshot
	}{sm.LastReviewedAt, sm.Difficulty, sm.Interval, sm.Historical})
}

// UnmarshalJSON implements json.Unmarshaller for Supermemo2Plus
func (sm *Supermemo2Plus) UnmarshalJSON(b []byte) error {
	payload := &struct {
		LastReviewedAt time.Time
		Difficulty     float64
		Interval       float64
		Historical     []IntervalSnapshot
	}{}

	if err := json.Unmarshal(b, payload); err != nil {
		return err
	}

	sm.LastReviewedAt = payload.LastReviewedAt
	sm.Difficulty = payload.Difficulty
	sm.Interval = payload.Interval
	sm.Historical = payload.Historical
	return nil
}
