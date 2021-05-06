// script for cleaning everything up once we've run the whole data pipeline, so
// as to ensure that we don't have a huge amount of synthetic data or previous days
// of data building up and crashing the server.

package main

import (
	"github.com/htried/wiki-diff-privacy/wdp"
	"log"
	"time"
	"os"
)

func main() {
	start := time.Now()
	// get a connection to the db
	db, err := wdp.DBConnection()
	if err != nil {
		log.Printf("Error %s when getting db connection", err)
		return
	}
	defer db.Close()
	log.Printf("Successfully connected to database")

	var yesterday = time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	if os.Args[1] != "data" && os.Args[1] != "output" {
		log.Printf("Error: incorrect database specified")
		return
	}

	err = wdp.DropOldData(db, os.Args[1], yesterday)
	if err != nil {
		log.Printf("Error %s when dropping synthetic data", err)
		return
	}

	log.Printf("Time to clean up database %s: %v seconds\n", os.Args[1], time.Now().Sub(start).Seconds())
}
