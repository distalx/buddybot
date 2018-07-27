package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

var (
	botToken   string
	usrToken   string
	reqSecret  string
	dbTable    string
	dbRegion   string
	uid        string
	tid        string
	adminGroup string
	db         *dynamodb.DynamoDB
)

// Handler is called on each inbound request. It receives an APIGatewayProxyRequest and returns
// an APIGatewayProxyResponse. Returning an error will indicate that the Lambda function has
// failed to execute successfully.
func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Println("INFO: request received", req.HTTPMethod, req.Path)

	// we expect all requests to be POST requests
	if req.HTTPMethod != http.MethodPost {
		fmt.Println("INFO: invalid method", req.HTTPMethod)
		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
		return resp, nil
	}

	// we expect Slack headers to be populated
	ts := req.Headers["X-Slack-Request-Timestamp"]
	if len(ts) < 10 {
		fmt.Println("INFO: invalid/no 'X-Slack-Request-Timestamp' header specified")
		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
		return resp, nil
	}

	sig := req.Headers["X-Slack-Signature"]
	if len(sig) < 3 {
		fmt.Println("invalid/no 'X-Slack-Signature' header specified")
		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
		return resp, nil
	}

	// validate the request signature is correct before handling the event
	valid := CheckHMAC(req.Body, ts, sig[3:], reqSecret)
	if valid != true {
		fmt.Println("failed to validate request signature")
		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
		return resp, nil
	}

	// get the admin channel if we don't have it already
	if adminGroup == "" {
		api := slack.New(usrToken)
		grps, err := api.GetGroups(true)
		if err != nil {
			fmt.Println("ERROR: unable to get bot details:", err)
			os.Exit(1)
		}
		for _, g := range grps {
			if g.NameNormalized == "admins" {
				adminGroup = g.ID
			}
		}
	}

	// parse the form to get the payload
	v, err := url.ParseQuery(req.Body)
	if err != nil {
		fmt.Println("failed to parse request body:", err)
		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
		return resp, nil
	}

	action, err := slackevents.ParseActionEvent(v.Get("payload"), slackevents.OptionVerifyToken(&NullComparator{}))
	if err != nil {
		fmt.Println("failed to parse request body:", err)
		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
		return resp, nil
	}

	switch action.CallbackId {
	case "flag":
		fmt.Println("INFO: message reported")
		fmt.Println("INFO: message:", action.OriginalMessage.Text)
		fmt.Println("INFO: flagged by:", action.User.Id)
		fmt.Println("INFO: admin channel:", adminGroup)

		api := slack.New(botToken)

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

		_, err := api.PostEphemeral(action.Channel.Id, action.User.Id,
			slack.MsgOptionPostEphemeral2(action.User.Id),
			slack.MsgOptionAttachments(attachment),
			slack.MsgOptionText("Message successfully flagged!", false),
		)
		if err != nil {
			fmt.Println("failed to notify reporter that message was flagged:", err)
			resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
			return resp, nil
		}

		// notify the admin channel
		fmt.Println("INFO: notifying admins:")
		api = slack.New(botToken)
		params := slack.PostMessageParameters{
			Username: uid,
			AsUser:   true,
			Markdown: true,
		}
		notification := fmt.Sprintf("Message flagged:\n>%s\n", action.OriginalMessage.Text)
		ts, _, err = api.PostMessage(adminGroup, notification, params)
		if err != nil {
			log.Println("ERROR: unable to post message:", err)
		}
		fmt.Println("INFO: admins notified:", ts)

	default:
		fmt.Println("INFO: unhandled action:", action.CallbackId)
		fmt.Printf("%+v\n", action)
	}

	resp := events.APIGatewayProxyResponse{StatusCode: http.StatusAccepted}
	return resp, nil
}

func main() {

	// Credentials are stored in the parameter store and not in environment variables if we fail
	// to retrieve these we terminate the application which indicates to Lambda that we have
	// failed to start successfully.
	err := requestParameters()
	if err != nil {
		fmt.Println("ERROR: unable to retrieve parameters:", err)
		os.Exit(1)
	}

	// Deployment details are stored in environment variables. If these are not set then we fail
	// to indicate that we don't have the required configuration to continue.
	dbTable = os.Getenv("BUDDYBOT_TABLE")
	if dbTable == "" {
		fmt.Println("ERROR: BUDDYBOT_TABLE is not set, please check the deployment configuration")
		os.Exit(1)
	}
	dbRegion = os.Getenv("BUDDYBOT_REGION")
	if dbRegion == "" {
		fmt.Println("ERROR: BUDDYBOT_REGION is not set, please check the deployment configuration")
		os.Exit(1)
	}

	// Create a new database session
	s, err := session.NewSession(&aws.Config{Region: aws.String(dbRegion)})
	if err != nil {
		fmt.Println("ERROR: unable to create database session:", err)
		os.Exit(1)
	}
	db = dynamodb.New(s)

	// Get bot details
	api := slack.New(botToken)
	atr, err := api.AuthTest()
	if err != nil {
		fmt.Println("ERROR: unable to get bot details:", err)
		os.Exit(1)
	}

	uid = atr.UserID
	tid = atr.TeamID

	// Tell Lamda we are ready to start accepting requests. This call blocks and does not return.
	lambda.Start(handler)
}

// requestParameters retrieves parameters from the AWS parameter store. It returns an error if
// any of the required parameters are undefined.
func requestParameters() error {
	svc := ssm.New(session.New())
	botTokenKey := "buddybot-botToken"
	usrTokenKey := "buddybot-usrToken"
	reqSecretKey := "buddybot-reqSecret"

	decrypt := true
	paramsIn := ssm.GetParametersInput{
		Names:          []*string{&botTokenKey, &usrTokenKey, &reqSecretKey},
		WithDecryption: &decrypt,
	}

	paramsOut, err := svc.GetParameters(&paramsIn)
	if err != nil {
		return err
	}
	params := make(map[string]string, len(paramsOut.Parameters))
	for _, p := range paramsOut.Parameters {
		params[*p.Name] = *p.Value
	}

	botToken = params[botTokenKey]
	if botToken == "" {
		return fmt.Errorf("%s is undefined", botTokenKey)
	}

	usrToken = params[usrTokenKey]
	if usrToken == "" {
		return fmt.Errorf("%s is undefined", usrTokenKey)
	}

	reqSecret = params[reqSecretKey]
	if reqSecret == "" {
		return fmt.Errorf("%s is undefined", reqSecretKey)
	}

	return nil
}

// CheckHMAC reports whether msgHMAC is a valid HMAC tag for msg.
func CheckHMAC(body, timestamp, msgHMAC, key string) bool {
	msg := "v0:" + timestamp + ":" + body
	hash := hmac.New(sha256.New, []byte(key))
	hash.Write([]byte(msg))

	expectedKey := hash.Sum(nil)
	actualKey, _ := hex.DecodeString(msgHMAC)
	return hmac.Equal(expectedKey, actualKey)
}

// NullComparator is a dummy comparator that allows us to define an
// empty Verify method
type NullComparator struct{}

// Verify always returns true, overriding the default token verification
// method. This is acceptable as we implement a separate check to confirm
// the validity of the request signature.
func (c NullComparator) Verify(string) bool {
	return true
}
