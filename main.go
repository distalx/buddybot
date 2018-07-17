package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/nlopes/slack/slackevents"
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

	eChan := make(chan slackevents.EventsAPIEvent)
	go handleEvents(eChan)

	Routes(eChan)
	http.ListenAndServe(":3000", nil)
}

// Routes sets up the routes for our web service.
func Routes(eChan chan slackevents.EventsAPIEvent) {
	http.HandleFunc("/events-endpoint", func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(r.Body)
		if err != nil {
			log.Println("no request body received")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body := buf.String()

		event, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&NullComparator{}))
		if err != nil {
			log.Println("failed to parse event:", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// process the slack event
		log.Println("event.Type:", event.Type)
		switch event.Type {
		case slackevents.URLVerification:
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(r.Challenge))
			return

		case slackevents.CallbackEvent:
			eChan <- event
			w.WriteHeader(http.StatusAccepted)

		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	})
}

func handleEvents(eChan chan slackevents.EventsAPIEvent) {
	for e := range eChan {
		log.Println("handling:", e.Type)
	}
	log.Println("terminating event handler")
}
