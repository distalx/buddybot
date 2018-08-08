package main

import (
	"fmt"
	"log"
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

		a, err := b.ParseAction(req)
		if err != nil {
			fmt.Println("WARN: failed to parse request:", err)
			resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
			return resp, nil
		}

		switch a.CallbackId {
		case "flag":

			// get the access tokens
			botToken, botUserToken, botUser, err := b.RetrieveTokens(a.Team.Id)
			if err != nil {
				fmt.Println("WARN: unable to retrieve team access token:", err)
				resp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
				return resp, nil
			}

			// get the admin channel
			adminGroup := ""
			api := slack.New(botUserToken)
			grps, err := api.GetGroups(true)
			if err != nil {
				fmt.Println("WARN: unable to retrieve list of groups:", err)
				resp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
				return resp, nil
			}
			for _, g := range grps {
				if g.NameNormalized == "admins" {
					adminGroup = g.ID
				}
			}

			fmt.Println("INFO: message reported")
			fmt.Println("INFO: message:", a.OriginalMessage.Text)
			fmt.Println("INFO: flagged by:", a.User.Id)
			fmt.Println("INFO: admin channel:", adminGroup)

			api = slack.New(botToken)

			attachment := slack.Attachment{
				Text:       "Let us know why you've flagged this message.",
				Color:      "#f9a41b",
				CallbackID: "flag_reason",
				Actions: []slack.AttachmentAction{
					{
						Name: "select",
						Type: "select",
						Options: []slack.AttachmentActionOption{
							{
								Text:  "spam",
								Value: "spam",
							},
							{
								Text:  "negativity",
								Value: "negativity",
							},
							{
								Text:  "abuse",
								Value: "abuse",
							},
							{
								Text:  "fast response",
								Value: "fast response",
							},
						},
					},
					{
						Name:  "cancel",
						Text:  "Cancel",
						Type:  "button",
						Style: "danger",
					},
				},
			}

			_, err = api.PostEphemeral(a.Channel.Id, a.User.Id,
				slack.MsgOptionPostEphemeral2(a.User.Id),
				slack.MsgOptionAttachments(attachment),
				slack.MsgOptionText("Message successfully flagged!", false),
			)
			if err != nil {
				fmt.Println("failed to notify reporter that message was flagged:", err)
				resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
				return resp, nil
			}

			// notify the admin channel
			api = slack.New(botToken)
			params := slack.PostMessageParameters{
				Username: botUser,
				AsUser:   true,
				Markdown: true,
			}
			notification := fmt.Sprintf("Message flagged:\n>%s\n", a.OriginalMessage.Text)
			ts, _, err := api.PostMessage(adminGroup, notification, params)
			if err != nil {
				log.Println("ERROR: unable to post message:", err)
			}
			fmt.Println("INFO: admins notified:", ts)

		default:
			fmt.Println("INFO: unhandled action:", a.CallbackId)
			fmt.Printf("%+v\n", a)
		}

		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusAccepted}
		return resp, nil
	}
}
