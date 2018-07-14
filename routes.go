package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"

	"github.com/nlopes/slack/slackevents"
)

// Routes sets up the routes for our web service.
func Routes() {
	http.HandleFunc("/events-endpoint", EventHandler)
}

// EventHandler handles inbound events posted by Slack
func EventHandler(w http.ResponseWriter, r *http.Request) {

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

	msg := "v0:" + ts + ":" + body

	valid := CheckHMAC(msg, sig[3:], secret)
	if valid != true {
		log.Println("failed to validate request signature")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&NullComparator{}))
	if err != nil {
		log.Println("failed to parse event:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case slackevents.URLVerification:
		log.Println("event type:", event.Type)
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
		switch ev := event.InnerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			log.Println("message text:", ev.Text)
			log.Println("message subtype:", ev.SubType)
		default:
			log.Println("unhandled message type:", event.InnerEvent.Type)
		}

	default:
		log.Println("event type:", event.Type)
		w.WriteHeader(http.StatusAccepted)
	}
}

// CheckHMAC reports whether msgMAC is a valid HMAC tag for msg.
func CheckHMAC(msg, msgMAC, key string) bool {
	hash := hmac.New(sha256.New, []byte(key))
	hash.Write([]byte(msg))

	expectedKey := hash.Sum(nil)
	actualKey, _ := hex.DecodeString(msgMAC)
	return hmac.Equal(expectedKey, actualKey)
}

type NullComparator struct{}

func (c NullComparator) Verify(string) bool {
	return true
}
