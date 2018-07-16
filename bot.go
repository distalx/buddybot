package main

import (
	"fmt"
	"log"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

func handleEvent(event slackevents.EventsAPIEvent) error {
	log.Println("---")
	log.Println("event.Type:", event.Type)
	log.Println("event.InnerEvent.Type:", event.InnerEvent.Type)

	// We are only interested in the MessageEvent
	switch ev := event.InnerEvent.Data.(type) {
	case *slackevents.MessageEvent:
		log.Printf("%+v\n", ev)
		err := scanMessage(ev)
		if err != nil {
			return err
		}

	default:
		log.Println("unhandled")
	}

	fmt.Println()
	return nil
}

func scanMessage(e *slackevents.MessageEvent) error {
	switch e.SubType {
	default:
		log.Println("e.SubType:", e.SubType)
		api := slack.New(token)
		at, err := api.AuthTest()
		if err != nil {
			return err
		}
		log.Println("at.UserID:", at.UserID)

		// TODO: Don't process messages from ourselves

		// postParams := slack.PostMessageParameters{}
		//api.PostMessage(e.Channel, "Yes, hello.", postParams)
	}

	return nil
}
