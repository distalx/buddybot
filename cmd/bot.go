package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

type bot struct {
	botToken  string
	usrToken  string
	reqSecret string
	region    string
}

func newBot() (*bot, error) {
	b := new(bot)

	svc := ssm.New(session.New())

	decrypt := true
	paramsIn := ssm.GetParametersInput{
		Names: []*string{
			aws.String("buddybot-botToken"),
			aws.String("buddybot-usrToken"),
			aws.String("buddybot-reqSecret"),
		},
		WithDecryption: &decrypt,
	}

	paramsOut, err := svc.GetParameters(&paramsIn)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get parameters from AWS parameter store")
	}
	params := make(map[string]string, len(paramsOut.Parameters))
	for _, p := range paramsOut.Parameters {
		params[*p.Name] = *p.Value
	}

	b.botToken = params["buddybot-botToken"]
	if b.botToken == "" {
		return nil, errors.New("required parameter 'buddybot-botToken' is undefined")
	}

	b.usrToken = params["buddybot-usrToken"]
	if b.usrToken == "" {
		return nil, errors.New("required parameter 'buddybot-usrToken' is undefined")
	}

	b.reqSecret = params["buddybot-reqSecret"]
	if b.reqSecret == "" {
		return nil, errors.New("required parameter 'buddybot-reqSecret' is undefined")
	}

	b.region = os.Getenv("BUDDYBOT_REGION")
	if b.region == "" {
		return nil, errors.New("required environmentle  'BUDDYBOT_REGION' is undefined")
	}

	return b, nil
}

func (b *bot) handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// we expect all requests to be POST requests
	if req.HTTPMethod != http.MethodPost {
		fmt.Println("WARN: invalid method", req.HTTPMethod)
		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
		return resp, nil
	}

	// we expect Slack headers to be populated
	ts := req.Headers["X-Slack-Request-Timestamp"]
	if len(ts) < 10 {
		fmt.Println("WARN: invalid/no 'X-Slack-Request-Timestamp' header specified")
		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
		return resp, nil
	}

	sig := req.Headers["X-Slack-Signature"]
	if len(sig) < 3 {
		fmt.Println("WARN: invalid/no 'X-Slack-Signature' header specified")
		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
		return resp, nil
	}

	// validate the request signature is correct before handling the event
	valid := CheckHMAC(req.Body, ts, sig[3:], b.reqSecret)
	if valid != true {
		fmt.Println("WARN: failed to validate request signature")
		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
		return resp, nil
	}

	s, err := SlashCommandParse(req.Body)
	if err != nil {
		fmt.Println("WARN: failed to parse request body:", err)
		resp := events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}
		return resp, nil
	}

	switch s.Command {

	case "/ping":
		fmt.Println("INFO: command received:", s.Command)
		fmt.Println("INFO: sent by:", s.TeamID, s.UserID, "(", s.UserName, ")")
		api := slack.New(b.botToken)
		_, err := api.PostEphemeral(s.ChannelID, s.UserID,
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

// SlashCommandParse is a local implementation of the slack.SlashCommandParse
// function. We use a local implementation as the slack package implementation
// requires access to the http.Request. Inside a Lambda function We have access
// to the body but not the original request.
func SlashCommandParse(r string) (slack.SlashCommand, error) {
	s := slack.SlashCommand{}

	v, err := url.ParseQuery(r)
	if err != nil {
		return s, err
	}

	s.Token = v.Get("token")
	s.TeamID = v.Get("team_id")
	s.TeamDomain = v.Get("team_domain")
	s.EnterpriseID = v.Get("enterprise_id")
	s.EnterpriseName = v.Get("enterprise_name")
	s.ChannelID = v.Get("channel_id")
	s.ChannelName = v.Get("channel_name")
	s.UserID = v.Get("user_id")
	s.UserName = v.Get("user_name")
	s.Command = v.Get("command")
	s.Text = v.Get("text")
	s.ResponseURL = v.Get("response_url")
	s.TriggerID = v.Get("trigger_id")
	return s, nil
}
