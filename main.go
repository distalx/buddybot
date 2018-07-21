package main

import (
	"log"
	"net/http"
	"os"

	"github.com/billglover/buddybot/plusplus"
	"github.com/nlopes/slack/slackevents"
)

var (
	authToken string
	userToken string
	secret    string
	port      string
	evtChan   chan slackevents.EventsAPIEvent
)

func init() {
	// TODO: There are mixed views on passing in credentials as environment variables.
	//       We should make a decision on whether this is the approach we want to take.
	//       We have made a conscious decision not to provide defaults to avoid
	//       accidental misconfiguration.
	authToken = os.Getenv("BUDDYBOT_TOKEN")
	if authToken == "" {
		log.Println("token must be provided by setting the BUDDYBOT_TOKEN EnvVar")
		os.Exit(1)
	}

	userToken = os.Getenv("BUDDYBOT_USER_TOKEN")
	if userToken == "" {
		log.Println("user token must be provided by setting the BUDDYBOT_USER_TOKEN EnvVar")
		os.Exit(1)
	}

	secret = os.Getenv("BUDDYBOT_SIGNING_SECRET")
	if secret == "" {
		log.Println("secret must be provided by setting the BUDDYBOT_SIGNING_SECRET EnvVar")
		os.Exit(1)
	}

	port = os.Getenv("BUDDYBOT_PORT")
	if port == "" {
		log.Println("port must be provided by setting the BUDDYBOT_PORT EnvVar")
		os.Exit(1)
	}

	evtChan = make(chan slackevents.EventsAPIEvent)
}

func main() {
	bb, _ := plusplus.New(authToken, userToken)
	go bb.Start(evtChan)

	Routes()
	http.ListenAndServe(":"+port, nil)
}
