package ui

import (
	"fmt"
	"strings"

	"github.com/komuw/leaf"
	runewidth "github.com/mattn/go-runewidth"
	termbox "github.com/nsf/termbox-go"
)

type step int

const (
	stepAnswering step = iota
	stepScore
	stepFinished
)

type align int

const (
	alignLeft align = iota
	alignCenter
	alignRight
)

// TUI implements terminal UI.
type TUI struct {
	deckName    string
	userInput   []rune
	step        step
	prevResult  bool
	prevCorrect string
}

// NewTUI construct a new TUI instance.
func NewTUI(deckName string) *TUI {
	return &TUI{deckName: deckName, userInput: make([]rune, 0)}
}

// Render renders current ui state using termbox.
func (ui *TUI) Render(s *SessionState) error {
	if s.Total == 0 {
		ui.step = stepFinished
	}

	ui.draw(s)

	for {
		ev := termbox.PollEvent()
		switch ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyEsc {
				return nil
			}

			if ui.step == stepFinished {
				break
			}

			if ui.step == stepScore {
				var score leaf.ReviewScore
				if s.RatingType == leaf.RatingTypeSelf {
					switch ev.Ch {
					case '1':
						score = leaf.ReviewScoreAgain
					case '2':
						score = leaf.ReviewScoreHard
					case '3':
						score = leaf.ReviewScoreGood
					case '4':
						score = leaf.ReviewScoreEasy
					default:
						continue
					}
				} else {
					if ui.prevResult {
						score = leaf.ReviewScoreEasy
					} else {
						score = leaf.ReviewScoreAgain
					}
				}

				s.Advance(score)
				if s.session.Left() == 0 {
					ui.step = stepFinished
				} else {
					ui.step = stepAnswering
				}

				break
			}

			if ev.Key == termbox.KeyEnter {
				ui.prevCorrect = s.ResolveAnswer()
				ui.prevResult = ui.prevCorrect == string(ui.userInput)
				ui.step = stepScore
				ui.userInput = make([]rune, 0)
			} else if ev.Key == termbox.KeyBackspace || ev.Key == termbox.KeyBackspace2 {
				if len(ui.userInput) > 0 {
					ui.userInput = ui.userInput[:len(ui.userInput)-1]
				}
			} else {
				var ch rune
				if ev.Key == termbox.KeySpace {
					ch = ' '
				} else {
					ch = ev.Ch
				}

				ui.userInput = append(ui.userInput, ch)
			}
		case termbox.EventError:
			return ev.Err
		}

		ui.draw(s)
	}
}

func (ui *TUI) draw(s *SessionState) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault) // nolint: errcheck
	defer termbox.Flush()

	w, h := termbox.Size()

	write(fmt.Sprintf("    Deck: %s", ui.deckName), 1, 1, 0, 0, 0)
	write(fmt.Sprintf("Progress: %d/%d", s.Total-s.Left, s.Total), 1, 2, 0, 0, 0)

	if ui.step == stepFinished {
		write("no more cards!", w/2, h/2-4, alignCenter, termbox.ColorGreen, 0)
		return
	}

	write(s.Question, w/2, h/2-4, alignCenter, termbox.ColorYellow|termbox.AttrBold, 0)
	if s.RatingType == leaf.RatingTypeSelf {
		ui.drawSelfRater(s)
	} else {
		ui.drawAutoRater(s)
	}
}

func (ui *TUI) drawAutoRater(s *SessionState) {
	w, h := termbox.Size()
	write("(type answer below)", w/2, h/2-3, alignCenter, 0, 0)

	x := (w / 2) - (s.AnswerLen / 2)
	inputBox := []rune{}
	for i := 0; i < s.AnswerLen; i++ {
		inputBox = append(inputBox, '_')
	}
	write(string(inputBox)+string('⏎'), x, h/2, 0, termbox.ColorWhite, 0)

	switch ui.step {
	case stepAnswering:
		input := strings.Replace(string(ui.userInput), " ", "␣", -1)
		write(input, x, h/2, 0, termbox.ColorGreen, 0)
	case stepScore:
		if ui.prevResult {
			write("✓", w/2, (h/2)+2, alignCenter, termbox.ColorGreen|termbox.AttrBold, 0)
		} else {
			write("✕", w/2, (h/2)+2, alignCenter, termbox.ColorRed|termbox.AttrBold, 0)
			write(ui.prevCorrect, w/2, (h/2)+3, alignCenter, termbox.ColorWhite, 0)
		}
	}
}

func (ui *TUI) drawSelfRater(s *SessionState) {
	w, h := termbox.Size()
	write("(select option below)", w/2, h/2-3, alignCenter, 0, 0)

	x := (w / 2) - (s.AnswerLen / 2)

	switch ui.step {
	case stepAnswering:
		write(" Show Answer: Enter ", x-9, h/2, 0, termbox.ColorMagenta, termbox.ColorWhite)
	case stepScore:
		write(ui.prevCorrect, w/2, h/2, alignCenter, termbox.ColorGreen, 0)
		scores := []string{" Again: 1 ", " Hard: 2 ", " Good: 3 ", " Easy: 4 "}
		for idx, score := range scores {
			scoreX := (w / 2) - 16
			for _, prev := range scores[0:idx] {
				scoreX += len(prev) + 1
			}
			write(score, scoreX, h/2+2, alignCenter, termbox.ColorMagenta, termbox.ColorWhite)
		}
	}
}

func write(text string, x, y int, align align, fg, bg termbox.Attribute) {
	var xOffset int
	switch align {
	case alignLeft:
		xOffset = x
	case alignCenter:
		xOffset = x - runewidth.StringWidth(text)/2
	case alignRight:
		xOffset = x - runewidth.StringWidth(text)
	}

	for _, c := range text {
		termbox.SetCell(xOffset, y, c, fg, bg)
		xOffset += runewidth.RuneWidth(c)
	}
}
