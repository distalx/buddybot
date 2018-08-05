package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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
			token, err := b.RetrieveToken(s.TeamID)
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
