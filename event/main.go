package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/billglover/buddybot/bot"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

func main() {
	b, err := bot.New()
	if err != nil {
		fmt.Println("ERROR: unable to initiate the bot:", err)
		os.Exit(1)
	}

	lambda.Start(handler(b))
}

func handler(b *bot.SlackBot) bot.APIHandler {

	return func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

		// parse the event
		e, err := b.ParseEvent(req)
		if err != nil {
			fmt.Println("WARN: failed to parse request:", err)
			resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
			return resp, nil
		}

		switch e.Type {

		// We must respond to URLVerification events to allow us to add the endpoint in
		// the Slack App.
		case slackevents.URLVerification:
			fmt.Println("INFO: event type:", e.Type)
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(req.Body), &r)
			if err != nil {
				fmt.Println("WARN: failed to parse challenge body:", err)
				resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
				return resp, nil
			}

			resp := events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
				Body:       r.Challenge,
				Headers:    map[string]string{"Content-Type": "text"},
			}
			return resp, nil

		// The CallbackEvent type contains inner events which we need to differentiate
		// before we can do anything useful.
		case slackevents.CallbackEvent:
			switch ev := e.InnerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:

				api := slack.New(b.BotToken)

				plusUsers := identifyPlusPlus(ev.Text)

				for _, u := range plusUsers {
					params := slack.PostMessageParameters{}

					if u == ev.User {
						reply := fmt.Sprintf("No <@%s>, try patting yourself on the back instead.", u)
						_, _, err := api.PostMessage(ev.Channel, reply, params)
						if err != nil {
							fmt.Println("WARN: unable to post message:", err)
						}
						break
					}

					reply := fmt.Sprintf("Congrats <@%s>! Score now at %d :smile:", u, 0)
					_, _, err = api.PostMessage(ev.Channel, reply, params)
					if err != nil {
						fmt.Println("WARN: unable to post message:", err)
					}
				}
			}

		default:
			fmt.Println("INFO: unhandled event", e.Type)
		}

		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusAccepted}
		return resp, nil
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
