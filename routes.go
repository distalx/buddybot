package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/nlopes/slack/slackevents"
)

// EventHandler handles inbound events posted by Slack
func eventHandler(w http.ResponseWriter, r *http.Request) {

	// we expect all requests to be POST requests
	if r.Method != http.MethodPost {
		log.Println("invalid method", r.Method, "expected", http.MethodPost)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// we expect Slack headers to be populated
	ts := r.Header.Get("X-Slack-Request-Timestamp")
	if len(ts) < 10 {
		log.Println("invalid/no 'X-Slack-Request-Timestamp' header specified")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("X-Slack-Signature")
	if len(sig) < 3 {
		log.Println("invalid/no 'X-Slack-Signature' header specified")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// we expect all requests to contain a body
	if r.ContentLength == 0 {
		log.Println("no request body received")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		log.Println("no request body received")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body := buf.String()

	// validate the request signature is correct before handling the event
	valid := CheckHMAC(body, ts, sig[3:], secret)
	if valid != true {
		log.Println("failed to validate request signature")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&NullComparator{}))
	if err != nil {
		log.Println("failed to parse request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {

	// Handle slack URLVerification by responding to the challenge request
	case slackevents.URLVerification:
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text")
		w.Write([]byte(r.Challenge))

	// Handle all CallbackEvent events asynchronously to ensure we respond
	// within the Slack web-hook timeout.
	case slackevents.CallbackEvent:
		evtChan <- event
		w.WriteHeader(http.StatusAccepted)

	// We don't handle any other types of event yet.
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

// Routes sets up the routes for our web service.
func Routes() {
	http.HandleFunc("/events-endpoint", eventHandler)
}
