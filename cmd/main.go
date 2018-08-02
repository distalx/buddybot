package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/billglover/buddybot/bot"
	"github.com/nlopes/slack"
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

		// parse the slash command
		s, err := b.ParseSlashCommand(req)
		if err != nil {
			fmt.Println("failed to parse request:", err)
			resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
			return resp, nil
		}

		// handle the individual slash commands
		// TODO: handle these asynchronously so we don't block the response to Slack
		switch s.Command {

		case "/ping":
			fmt.Println("INFO: command received:", s.Command)
			fmt.Println("INFO: sent by:", s.TeamID, s.UserID, "(", s.UserName, ")")

			// retrieve the appropriate bot token
			token, err := retrieveToken(b, s.TeamID)
			if err != nil {
				fmt.Println("WARN: unable to retrieve access token:", err)
				resp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
				return resp, nil
			}

			api := slack.New(token)
			_, err = api.PostEphemeral(s.ChannelID, s.UserID,
				slack.MsgOptionPostEphemeral2(s.UserID),
				slack.MsgOptionText("Pong!", false),
			)
			if err != nil {
				fmt.Println("WARN: failed to respond to ping:", err)
				resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
				return resp, nil
			}

		default:
			fmt.Println("INFO: unknown command sent:", s.Command)
		}

		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusAccepted}
		return resp, nil
	}
}

// AuthRecord represents the access token we store in DynamoDB for
// every authenticated workspace.
type AuthRecord struct {
	UID            string `json:"uid"`
	AccessToken    string `json:"access_token"`
	Scope          string `json:"scope"`
	UserID         string `json:"user_id"`
	TeamName       string `json:"team_name"`
	TeamID         string `json:"team_id"`
	BotUserID      string `json:"bot_user_id"`
	BotAccessToken string `json:"bot_access_token"`
}

func retrieveToken(b *bot.SlackBot, teamID string) (string, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(b.Region)})
	if err != nil {
		return "", err
	}

	ddb := dynamodb.New(sess)

	input := &dynamodb.GetItemInput{
		TableName: aws.String(b.AuthTable),
		Key:       map[string]*dynamodb.AttributeValue{"uid": {S: aws.String(teamID)}},
	}

	result, err := ddb.GetItem(input)
	if err != nil {
		return "", err
	}

	item := AuthRecord{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
	if err != nil {
		return "", err
	}

	return item.BotAccessToken, nil
}
