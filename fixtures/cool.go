package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"time"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"github.com/sanity-io/litter"
)

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

////////////////////////////////////////////// SUPERMEMO //////////////////////////////////////////////

// Deck represents a named collection of the cards to review.
type Deck struct {
	Name      string
	Cards     []Card
	Algorithm string //SRS
	filename  string
}

// Card represents a single card in a Deck.
type Card struct {
	Question string
	FullCard []byte
	Filename string
	//Algorithm is probably not needed if
	// - we only ever have one algo
	// - rater is always `self`
	Algorithm Supermemo2
}

// NextReviewAt returns next review timestamp for a card.
func (card *Card) NextReviewAt() time.Time {
	return card.Algorithm.LastReviewedAt.Add(time.Duration(24*card.Algorithm.Interval) * time.Hour)
}

// Advance advances supermemo state for a card.
func (card *Card) Advance(rating float64) float64 {
	card.Algorithm.Total++
	card.Algorithm.LastReviewedAt = time.Now()

	card.Algorithm.Easiness += 0.1 - (1-rating)*(0.4+(1-rating)*0.5)
	card.Algorithm.Easiness = math.Max(card.Algorithm.Easiness, 1.3)

	const ratingSuccess = 0.6
	interval := 1.0
	if rating >= ratingSuccess {
		if card.Algorithm.Total == 2 {
			interval = 6
		} else if card.Algorithm.Total > 2 {
			interval = math.Round(card.Algorithm.Interval * card.Algorithm.Easiness)
		}
		card.Algorithm.Correct++
	} else {
		card.Algorithm.Correct = 0
	}

	// if card.Algorithm.Historical == nil {
	// 	card.Algorithm.Historical = make([]IntervalSnapshot, 0)
	// }
	// card.Algorithm.Historical = append(
	// 	card.Algorithm.Historical,
	// 	IntervalSnapshot{time.Now().Unix(), card.Algorithm.Interval, card.Algorithm.Easiness},
	// )

	card.Algorithm.Interval = interval
	return interval
}

func main() {
	filename := "/Users/komuw/mystuff/leaf/fixtures/cool.md"
	md, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("err ", err)
	}
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	parser := parser.NewWithExtensions(extensions)

	mainNode := parser.Parse(md)

	newCard := &Card{
		FullCard:  md,
		Filename:  filename,
		Algorithm: NewSupermemo2(),
	}
	for k, child := range mainNode.GetChildren() {
		fmt.Println("\nkkk:", k)
		identifyNode(child, newCard)
	}

	fmt.Println("newCard")
	litter.Dump(newCard)

	fmt.Println("NextReviewAt() 1: ", newCard.NextReviewAt())
	litter.Dump(newCard)

	// review & rate a card
	newCard.Advance(0.5)
	fmt.Println("NextReviewAt() 2: ", newCard.NextReviewAt())
	litter.Dump(newCard)

}

func identifyNode(node ast.Node, card *Card) {
	switch thisNode := node.(type) {
	case *ast.HTMLBlock:
		// for metadata
		fmt.Println("HTMLBlock.Literal", string(thisNode.Literal))

	case *ast.Heading:
		// for question
		fmt.Println("HeadingID:\n", thisNode.HeadingID)
		card.Question = thisNode.HeadingID
	case *ast.CodeBlock:
		// for answer
		fmt.Println("codeBlock.Info", string(thisNode.Info))
		fmt.Println("codeBlock.Literal:\n", string(thisNode.Literal))
	default:
		// unknown
		fmt.Println("Unknown node ", thisNode)
		// litter.Dump(node)
	}

}
