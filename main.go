package main

import (
	"log"
	"net/http"
	"os"
)

var token string
var secret string

func main() {
	println("BuddyBot")

	// TODO: There are mixed views on passing in credentials as environment variables.
	//       We should make a decision on whether this is the approach we want to take.
	//       We have made a conscious decision not to provide defaults to avoid
	//       accidental misconfiguration.
	token = os.Getenv("BUDDYBOT_TOKEN")
	if token == "" {
		log.Println("token must be provided by setting the BUDDYBOT_TOKEN EnvVar")
		os.Exit(1)
	}

	secret = os.Getenv("BUDDYBOT_SIGNING_SECRET")
	if secret == "" {
		log.Println("secret must be provided by setting the BUDDYBOT_SIGNING_SECRET EnvVar")
		os.Exit(1)
	}

	Routes()
	http.ListenAndServe(":3000", nil)
}

// Routes sets up the routes for our web service.
func Routes() {
	http.HandleFunc("/events-endpoint", EventHandler)
}
