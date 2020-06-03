package main

import (
	log "log"
	os "os"

	authentication "github.com/dendrite2go/dendrite/src/pkg/authentication"
)

func main() {
	for _, password := range os.Args[1:] {
		log.Printf("Encoded password: %v", authentication.Encode(password))
	}
}
