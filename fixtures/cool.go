package main

import (
	"encoding/json"

	"fmt"
	"io/ioutil"
	"log"
	"math"
	"time"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"github.com/sanity-io/litter"

	"github.com/pkg/errors"
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
}

const attrName = "user.algo" // has to start with "user."

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

////////////////////////////////////////////// SUPERMEMO //////////////////////////////////////////////

// Card represents a single card in a Deck.
type Card struct {
	Version  uint32
	Question string
	// FileContents []byte
	FilePath  string
	Algorithm SRSalgorithm
}

// MarshalJSON implements json.Marshaler for Supermemo2
// func (c *Card) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(&struct {
// 		Version  uint32
// 		Question string
// 		// FileContents []byte
// 		FilePath  string
// 		Algorithm SRSalgorithm
// 	}{c.Version, c.Question, c.FilePath, c.Algorithm})
// }

// UnmarshalJSON implements json.Unmarshaler for Supermemo2
func (c *Card) UnmarshalJSON(b []byte) error {
	//var payload Card

	var objMap map[string]interface{}
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return errors.Wrapf(err, "unable to Unmarshal")
	}

	c.Version = uint32(objMap["Version"].(float64))
	c.Question = objMap["Question"].(string)
	c.FilePath = objMap["FilePath"].(string)

	var objMapAlgo = objMap["Algorithm"].(map[string]interface{})
	myAlg := NewSupermemo2()
	myAlg.Interval = objMapAlgo["Interval"].(float64)
	myAlg.Easiness = objMapAlgo["Easiness"].(float64)
	myAlg.Correct = int(objMapAlgo["Correct"].(float64))
	myAlg.Total = int(objMapAlgo["Total"].(float64))
	LastReviewedAtLayout := "2006-01-02T15:04:05Z07:00"
	ReviewedAt, err := time.Parse(LastReviewedAtLayout, objMapAlgo["LastReviewedAt"].(string))
	if err != nil {
		return errors.Wrapf(err, "unable to Parse LastReviewedAt")
	}
	myAlg.LastReviewedAt = ReviewedAt

	c.Algorithm = myAlg

	fmt.Println("card11:")
	litter.Dump(c)

	fmt.Println("payload:")
	litter.Dump(c)

	fmt.Println("card:")
	litter.Dump(c)

	return nil
}

func main() {
	filepath := "/home/komuw/mystuff/leaf/fixtures/akk.md"
	md, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalf("error: %+v", err)
	}
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	parser := parser.NewWithExtensions(extensions)
	mainNode := parser.Parse(md)
	question, err := getQuestion(mainNode)
	if err != nil {
		log.Fatalf("error: %+v", err)
	}
	cardAttribute, err := getExtendedAttrs(filepath)
	if err != nil {
		log.Fatalf("error: %+v", err)

	}
	fmt.Println("cardAttribute:")
	litter.Dump(string(cardAttribute))

	// if cardAttribute exists, then this is not a new card and we should
	// bootstrap the Algorithm to use from the cardAttribute
	// else, create a card with a new Algorithm
	card := Card{
		Version:   1,
		Question:  question,
		FilePath:  filepath,
		Algorithm: NewSupermemo2(),
	}
	if len(cardAttribute) > 0 {
		var crd Card

		err = json.Unmarshal(cardAttribute, &crd)
		if err != nil {
			log.Fatalf("error: %+v", err)
		}

		fmt.Println("card from file")
		litter.Dump(crd)

		card = crd
	}

	fmt.Println("card before death")
	litter.Dump(card)
	fmt.Println("NextReviewAt() 1: ", card.Algorithm.NextReviewAt())
	// review and rate a card
	sm := card.Algorithm.Advance(0.8)
	card.Algorithm = sm
	fmt.Println("NextReviewAt() 2: ", card.Algorithm.NextReviewAt())

	// update the card attributes with new algo
	algoJson, err := json.Marshal(card)
	if err != nil {
		log.Fatalf("error: %+v", err)
	}
	err = setExtendedAttrs(filepath, algoJson)
	if err != nil {
		log.Fatalf("error: %+v", err)

	}

	fmt.Println("card when saving")
	litter.Dump(card)

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
	err := xattr.Set(filepath, attrName, algoJson)
	if err != nil {
		return errors.Wrapf(err, "unable to set extended file attributes")
	}
	return nil
}

func getExtendedAttrs(filepath string) ([]byte, error) {
	attribute, err := xattr.Get(filepath, attrName)
	if len(attribute) > 0 && err != nil {
		return []byte(""), errors.Wrapf(err, "unable to get extended file attributes")
	}
	return attribute, nil
}
