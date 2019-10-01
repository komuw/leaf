package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/komuw/leaf"
	"github.com/komuw/leaf/ui"
)

var (
	decks   = flag.String("decks", ".", "deck files location")
	db      = flag.String("db", "leaf.db", "stats database location")
	addr    = flag.String("addr", ":8000", "addr for Web UI")
	devMode = flag.Bool("dev", false, "use local dev assets")
)

func main() {
	flag.Parse()

	db, err := leaf.OpenBoltStore(*db)
	if err != nil {
		log.Fatal("Failed to open stats DB: ", err)
	}

	defer db.Close()

	dm, err := leaf.NewDeckManager(*decks, db, leaf.OutputFormatHTML)
	if err != nil {
		log.Fatal("Failed to initialise deck manager: ", err)
	}

	srv := ui.NewServer(dm)
	handler := srv.Handler(*devMode)
	fs := http.FileServer(http.Dir(*decks))
	handler.Handle("/images/", http.StripPrefix("/images", fs))

	log.Println("Serving HTTP on", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal("Failed to render: ", err)
	}
}
