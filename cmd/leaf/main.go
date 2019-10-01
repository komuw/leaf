package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/komuw/leaf"
	"github.com/komuw/leaf/ui"
	termbox "github.com/nsf/termbox-go"
)

var (
	decks = flag.String("decks", ".", "deck files location")
	db    = flag.String("db", "leaf.db", "stats database location")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [args] [stats|review] [deck_name]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Example: %s -decks ./fixtures review Hiragana\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Optional arguments:")
		flag.PrintDefaults()
	}
	flag.Parse()

	deckName := flag.Arg(1)
	if deckName == "" {
		log.Fatal("Missing deck name")
	}

	db, err := leaf.OpenBoltStore(*db)
	if err != nil {
		log.Fatal("Failed to open stats DB: ", err)
	}

	defer db.Close()

	dm, err := leaf.NewDeckManager(*decks, db, leaf.OutputFormatOrg)
	if err != nil {
		log.Fatal("Failed to initialise deck manager: ", err)
	}

	switch flag.Arg(0) {
	case "stats":
		stats, err := dm.DeckStats(deckName)
		if err != nil {
			log.Fatal("Failed to get card stats: ", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 5, 5, 5, ' ', 0)
		fmt.Fprintln(w, "Card\tStats")
		for _, s := range stats {
			stat, err := json.Marshal(s)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "%s\t%s\n", s.Question, stat)
		}
		w.Flush()
	case "review":
		session, err := dm.ReviewSession(deckName)
		if err != nil {
			log.Fatal("Failed to create review session: ", err)
		}

		if err := termbox.Init(); err != nil {
			log.Fatal("Failed to initialise tui: ", err)
		}
		defer termbox.Close()

		u := ui.NewTUI(deckName)

		if err := u.Render(ui.NewSessionState(session)); err != nil {
			log.Fatal("Failed to render: ", err)
		}
	default:
		log.Fatal("unknown command")
	}
}
