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

		a, err := b.ParseAction(req)
		if err != nil {
			fmt.Println("WARN: failed to parse request:", err)
			resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
			return resp, nil
		}

		switch a.CallbackId {
		case "flag":

			// Request access tokens
			botToken, botUserToken, botUser, err := b.RetrieveTokens(a.Team.Id)
			if err != nil {
				fmt.Println("WARN: unable to retrieve team access token:", err)
				resp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
				return resp, nil
			}

			// Searching for the admin channel requires the Bot User token rather than the Bot token.
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

			// Notify the reporter that we have received their report
			api = slack.New(botToken)
			_, err = api.PostEphemeral(a.Channel.Id, a.User.Id,
				slack.MsgOptionPostEphemeral2(a.User.Id),
				slack.MsgOptionText("This message has been flagged!\nWe'll review it against our Code of Conduct and take appropriate action. If we need more information, one of the admins will be in touch privately for more information.", false),
			)
			if err != nil {
				fmt.Println("WARN: failed to notify reporter that message was flagged:", err)
				resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
				return resp, nil
			}

			// Notify the original author that their message has been flagged
			api = slack.New(botToken)
			msgText := "This message that you posted has been flagged as potentially violating our Code of Conduct!\n> \"" + a.OriginalMessage.Text + "\"\n\nThe message may be removed or one of the admins may be in touch shortly to discuss this post. We know that not all CoC breaches are intentional, so please consider reviewing your post and notifying the thread of any changes."
			_, err = api.PostEphemeral(a.Channel.Id, a.OriginalMessage.User,
				slack.MsgOptionPostEphemeral2(a.User.Id),
				slack.MsgOptionText(msgText, false),
			)
			if err != nil {
				fmt.Println("WARN: failed to notify reporter that message was flagged:", err)
				resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
				return resp, nil
			}

			// Request a permalink to the flagged message
			linkParams := slack.GetPermalinkParameters{
				Channel: a.Channel.Id,
				Ts:      a.OriginalMessage.Timestamp,
			}
			permalink, err := api.GetPermalink(&linkParams)
			if err != nil {
				fmt.Println("ERROR: unable to get message permalink:", err)
			}

			// Get user and channel names to allow us to post a useful message in the admin channel
			reporter, err := api.GetUserInfo(a.User.Id)
			if err != nil {
				fmt.Println("ERROR: unable to get the reporter name:", err)
			}

			author, err := api.GetUserInfo(a.OriginalMessage.User)
			if err != nil {
				fmt.Println("ERROR: unable to get the author name:", err)
			}

			channel, err := api.GetChannelInfo(a.Channel.Id)
			if err != nil {
				fmt.Println("ERROR: unable to get the channel name:", err)
			}

			attachment := slack.Attachment{
				Title:     "Flagged message",
				TitleLink: permalink,
				Color:     "danger",
				Pretext:   "The message below has been flagged for a potential CoC violation",
				Fields: []slack.AttachmentField{
					slack.AttachmentField{Title: "Reporter", Value: reporter.Name, Short: true},
					slack.AttachmentField{Title: "Author", Value: author.Name, Short: true},
					slack.AttachmentField{Title: "Channel", Value: channel.Name, Short: true},
					slack.AttachmentField{Title: "Message", Value: a.OriginalMessage.Text, Short: false},
				},
			}
			msgParams := slack.PostMessageParameters{
				Username:    botUser,
				AsUser:      true,
				Markdown:    true,
				Attachments: []slack.Attachment{attachment},
			}

			_, _, err = api.PostMessage(adminGroup, "", msgParams)
			if err != nil {
				fmt.Println("ERROR: unable to post message:", err)
			}
			fmt.Println("INFO: message by", author.Name, "flagged by", reporter.Name)

		default:
			fmt.Println("INFO: unhandled action:", a.CallbackId)
			fmt.Printf("%+v\n", a)
		}

		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusAccepted}
		return resp, nil
	}
}
