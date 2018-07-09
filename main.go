package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/nlopes/slack"
)

func main() {
	println("BuddyBot")

	token := os.Getenv("BUDDYBOT_TOKEN")
	if token == "" {
		fmt.Println("Token must be provided by setting the BUDDYBOT_TOKEN EnvVar")
		os.Exit(1)
	}

	api := slack.New(token)
	//logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	//slack.SetLogger(logger)
	api.SetDebug(false)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {

		case *slack.ConnectedEvent:
			fmt.Println("Infos:", ev.Info)
			fmt.Println("Connection counter:", ev.ConnectionCount)
			rtm.SendMessage(rtm.NewOutgoingMessage("BuddyBot reporting for duty", "CBLPRTX3P"))

		case *slack.MessageEvent:
			fmt.Printf("message text: %v\n", ev.Text)
			fmt.Printf("%v\n", ev)
			users := identifyPlusPlus(ev.Text)
			for _, u := range users {
				msgParams := slack.PostMessageParameters{
					ThreadTimestamp: ev.EventTimestamp,
				}
				_, _, err := rtm.PostMessage(ev.Channel, fmt.Sprintf("Well done <@%s>!", u), msgParams)
				if err != nil {
					fmt.Println("unable to leave reply:", err)
				}
			}

		case *slack.RTMError:
			fmt.Println("Error:", ev.Error())

		case *slack.InvalidAuthEvent:
			fmt.Println("invalid credentials")
			return

		default:
			fmt.Println("unhandled event received:", msg.Type)
		}
	}
}

// IdentifyPlusPlus takes a message and returns a slice of users tagged for PlusPlus.
func identifyPlusPlus(msg string) []string {
	var users []string
	var re = regexp.MustCompile(`(?m)\<\@(\w+)\>\+\+`)
	for _, match := range re.FindAllStringSubmatch(msg, -1) {
		users = append(users, string(match[1]))
	}
	return users
}
