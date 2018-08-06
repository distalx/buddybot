package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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
			cbe := e.Data.(*slackevents.EventsAPICallbackEvent)
			switch ev := e.InnerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:

				// retrieve the appropriate bot token
				token, err := b.RetrieveToken(cbe.TeamID)
				if err != nil {
					fmt.Println("WARN: unable to retrieve access token:", err)
					resp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
					return resp, nil
				}

				api := slack.New(token)

				plusUsers := identifyPlusPlus(ev.Text)

				for _, u := range plusUsers {
					params := slack.PostMessageParameters{}

					if u == ev.User {
						reply := fmt.Sprintf("No <@%s>, try patting yourself on the back instead :stuck_out_tongue_closed_eyes:", u)
						_, _, err := api.PostMessage(ev.Channel, reply, params)
						if err != nil {
							fmt.Println("WARN: unable to post message:", err)
						}
						break
					}

					// TODO: update score in database

					sess, err := session.NewSession(&aws.Config{Region: aws.String(b.Region)})
					if err != nil {
						fmt.Println("ERROR: unable to create session:", err)
						break
					}

					ddb := dynamodb.New(sess)

					input := &dynamodb.UpdateItemInput{
						ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":s": {N: aws.String("1")}},
						TableName:                 aws.String(b.ScoreTable),
						Key:                       map[string]*dynamodb.AttributeValue{"uid": {S: aws.String(cbe.TeamID + ":" + u)}},
						ReturnValues:              aws.String("UPDATED_NEW"),
						UpdateExpression:          aws.String("add score :s"),
					}

					v, err := ddb.UpdateItem(input)
					if err != nil {
						fmt.Println("ERROR: unable to update database:", err)
						break
					}

					fmt.Printf("INFO: %+v\n", v)
					fmt.Printf("INFO: %+v\n", v.Attributes["score"])

					record := new(int)
					err = dynamodbattribute.Unmarshal(v.Attributes["score"], record)

					if err != nil {
						fmt.Println("ERROR: unable to unmarshal return value:", err)
						break
					}

					reply := fmt.Sprintf("Congrats <@%s>! Score now at %d :smile:", u, *record)
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
