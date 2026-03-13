package main

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/internal/search"
	"log"
)

func main() {
	log.Printf("Starting search: %s -> %s on %s", arguments.FROM, arguments.TO, arguments.DATE)
	log.Printf("Looking for trains: %v, seat types: %v, min seats: %d",
		arguments.SPECIFIC_TRAIN_ARRAY, arguments.SEAT_TYPE_ARRAY, arguments.SEAT_COUNT)

	msg, found := search.PerformSearch(arguments.AUTH_FILE)
	if found {
		log.Println("Success!", msg)
	} else {
		log.Println("Search ended without finding seats.")
	}
}
