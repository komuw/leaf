package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"time"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"github.com/sanity-io/litter"

	"github.com/pkg/xattr"
)

// Deck represents a named collection of the cards to review.
type Deck struct {
	Name      string
	Cards     []Card
	Algorithm string //SRS
	filename  string
}

type SRSalgorithm interface {
	NextReviewAt() time.Time
	Advance(rating float64) SRSalgorithm

	MarshalJSON() ([]byte, error)
	UnmarshalJSON(b []byte) error
}

////////////////////////////////////////////// SUPERMEMO //////////////////////////////////////////////
// Supermemo2 calculates review intervals using SM2 algorithm
type Supermemo2 struct {
	LastReviewedAt time.Time
	Interval       float64
	Easiness       float64
	Correct        int
	Total          int
}

// NewSupermemo2 returns a new Supermemo2 instance
func NewSupermemo2() Supermemo2 {
	return Supermemo2{
		LastReviewedAt: time.Now(),
		Interval:       0,
		Easiness:       2.5,
		Correct:        0,
		Total:          0,
	}
}

// NextReviewAt returns next review timestamp for a card.
func (sm Supermemo2) NextReviewAt() time.Time {
	return sm.LastReviewedAt.Add(time.Duration(24*sm.Interval) * time.Hour)
}

// Advance advances supermemo state for a card.
func (sm Supermemo2) Advance(rating float64) SRSalgorithm {

	newSm := sm

	newSm.Total++
	newSm.LastReviewedAt = time.Now()

	newSm.Easiness += 0.1 - (1-rating)*(0.4+(1-rating)*0.5)
	newSm.Easiness = math.Max(newSm.Easiness, 1.3)

	const ratingSuccess = 0.6
	interval := 1.0
	if rating >= ratingSuccess {
		if newSm.Total == 2 {
			interval = 6
		} else if newSm.Total > 2 {
			interval = math.Round(newSm.Interval * newSm.Easiness)
		}
		newSm.Correct++
	} else {
		newSm.Correct = 0
	}
	newSm.Interval = interval

	return newSm
}

// MarshalJSON implements json.Marshaler for Supermemo2
func (sm Supermemo2) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		LastReviewedAt time.Time
		Interval       float64
		Easiness       float64
		Correct        int
		Total          int
	}{sm.LastReviewedAt, sm.Interval, sm.Easiness, sm.Correct, sm.Total})
}

// UnmarshalJSON implements json.Unmarshaler for Supermemo2
func (sm Supermemo2) UnmarshalJSON(b []byte) error {
	payload := &struct {
		LastReviewedAt time.Time
		Interval       float64
		Easiness       float64
		Correct        int
		Total          int
	}{}

	if err := json.Unmarshal(b, payload); err != nil {
		return err
	}

	sm.LastReviewedAt = payload.LastReviewedAt
	sm.Easiness = payload.Easiness
	sm.Interval = payload.Interval
	sm.Correct = payload.Correct
	sm.Total = payload.Total
	return nil
}

////////////////////////////////////////////// SUPERMEMO //////////////////////////////////////////////

// Card represents a single card in a Deck.
type Card struct {
	Question     string
	FileContents []byte
	FilePath     string
	Algorithm    SRSalgorithm
}

func main() {
	filepath := "/home/komuw/mystuff/leaf/fixtures/cool.md"
	md, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatal("error: ", err)
	}
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	parser := parser.NewWithExtensions(extensions)

	mainNode := parser.Parse(md)

	question, err := getQuestion(mainNode)
	if err != nil {
		log.Fatal("error: ", err)
	}

	card := Card{
		FileContents: md,
		FilePath:     filepath,
		Algorithm:    NewSupermemo2(),
		Question:     question,
	}

	fmt.Println("card")
	// // litter.Dump(card)

	fmt.Println("NextReviewAt() 1: ", card.Algorithm.NextReviewAt())
	// litter.Dump(card)

	// review and rate a card
	sm := card.Algorithm.Advance(0.8)
	card.Algorithm = sm
	fmt.Println("NextReviewAt() 2: ", card.Algorithm.NextReviewAt())
	// litter.Dump(card)

	algoJson, err := json.Marshal(card.Algorithm)
	if err != nil {
		log.Fatal("error: ", err)
	}
	fmt.Println("algoJson: ", algoJson)

	err = setExtendedAttrs(filepath, algoJson)
	if err != nil {
		log.Fatal("error: ", err)

	}

	th := card.Algorithm.(Supermemo2)
	err = json.Unmarshal(algoJson, &th)
	if err != nil {
		log.Fatal("error: ", err)
	}
	fmt.Println("theRealAlgo")
	litter.Dump(th)

	fmt.Println("th NextReviewAt() 2: ", th.NextReviewAt())

}

func getQuestion(node ast.Node) (string, error) {
	for _, child := range node.GetChildren() {
		switch thisNode := child.(type) {
		case *ast.Heading:
			question := thisNode.HeadingID
			return question, nil
		default:
			// unknown Node
		}
	}
	return "", errors.New("The markdown file does not contain a question")
}

func setExtendedAttrs(filepath string, algoJson []byte) error {
	const attrName = "user.algo" // has to start with "user."
	err := xattr.Set(filepath, attrName, algoJson)
	if err != nil {
		return fmt.Errorf("unable to set extended file attributes: %w", err)
	}
	return nil
}
