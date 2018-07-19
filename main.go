package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/billglover/buddybot/plusplus"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

var (
	token   string
	secret  string
	port    string
	evtChan chan slackevents.EventsAPIEvent
)

func init() {
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

	port = os.Getenv("BUDDYBOT_PORT")
	if port == "" {
		log.Println("port must be provided by setting the BUDDYBOT_PORT EnvVar")
		os.Exit(1)
	}

	evtChan = make(chan slackevents.EventsAPIEvent)
}

func main() {
	bb, _ := plusplus.New(token)
	go bb.Start(evtChan)

	time.AfterFunc(time.Second*10.0, func() {
		api := slack.New(token)
		params := slack.PostMessageParameters{
			Username: "UBLPTK0JH",
			AsUser:   true,
		}
		rc, rt, err := api.PostMessage("CBLPRTX3P", "BuddyBot reporting for duty!", params)
		log.Println("resp chan:", rc)
		log.Println("resp ts:", rt)
		log.Println("resp err:", err)
	})

	Routes()
	http.ListenAndServe(":"+port, nil)
}
