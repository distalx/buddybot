package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/billglover/buddybot/bot"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
	"github.com/pkg/errors"
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

		e, err := b.ParseEvent(req)
		if err != nil {
			fmt.Println("WARN: failed to parse request:", err)
			resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
			return resp, nil
		}

		switch e.Type {

		case slackevents.URLVerification:

			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(req.Body), &r)
			if err != nil {
				fmt.Println("WARN: failed to parse URL Verification challenge:", err)
				resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
				return resp, nil
			}

			resp := events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
				Body:       r.Challenge,
				Headers:    map[string]string{"Content-Type": "text"},
			}
			return resp, nil

		case slackevents.CallbackEvent:
			cbe := e.Data.(*slackevents.EventsAPICallbackEvent)

			switch ev := e.InnerEvent.Data.(type) {

			case *slackevents.AppMentionEvent:
				token, err := b.RetrieveToken(cbe.TeamID)
				if err != nil {
					fmt.Println("WARN: unable to retrieve team access token:", err)
					resp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
					return resp, nil
				}
				api := slack.New(token)

				plusUsers := identifyPlusPlus(ev.Text)

				for _, u := range plusUsers {
					params := slack.PostMessageParameters{}

					// Don't let users boost their own egos
					if u == ev.User {
						reply := fmt.Sprintf("No <@%s>, try patting yourself on the back instead :stuck_out_tongue_closed_eyes:", u)
						_, _, err := api.PostMessage(ev.Channel, reply, params)
						if err != nil {
							fmt.Println("WARN: unable to post message:", err)
						}
						break
					}

					score, err := incrementScore(b, cbe.TeamID, u)
					reply := fmt.Sprintf("Congrats <@%s>! Score now at %d :smile:", u, score)
					if err != nil {
						fmt.Println("WARN: unable to increment score:", err)
						reply = fmt.Sprintf("Congrats <@%s>! I was unable to update your score, so you'll have to accept this smile instead :smile:", u)
					}

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

// IncrementScore takes a team and a user an increments the score by one. It returns the
// new score or an error.
func incrementScore(b *bot.SlackBot, team, user string) (int, error) {

	score := 0

	sess, err := session.NewSession(&aws.Config{Region: aws.String(b.Region)})
	if err != nil {
		return score, errors.Wrap(err, "unable to create session")
	}

	ddb := dynamodb.New(sess)

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":s": {N: aws.String("1")}},
		TableName:                 aws.String(b.ScoreTable),
		Key:                       map[string]*dynamodb.AttributeValue{"uid": {S: aws.String(team + ":" + user)}},
		ReturnValues:              aws.String("UPDATED_NEW"),
		UpdateExpression:          aws.String("add score :s"),
	}

	v, err := ddb.UpdateItem(input)
	if err != nil {
		return score, errors.Wrap(err, "unable to update database")
	}

	err = dynamodbattribute.Unmarshal(v.Attributes["score"], &score)

	if err != nil {
		return score, errors.Wrap(err, "unable to unmarshal return value")
	}

	return score, nil
}
