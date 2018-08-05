package bot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
	"github.com/pkg/errors"
)

// APIHandler is a function signature for the AWS API Gatway Request handler
type APIHandler func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// SlackBot is an instance of the Slack bot
type SlackBot struct {
	ClientID     string
	ClientSecret string
	BotToken     string
	UsrToken     string
	ReqSecret    string
	Region       string
	AuthTable    string
	ScoreTable   string
}

// New returns an instance of a SlackBot. It retrieves credentials from the AWS Parameter Store
// and reads configuration from environment variables. If it is unable to retrieve the expected
// values it returns an error.
func New() (*SlackBot, error) {
	b := new(SlackBot)

	svc := ssm.New(session.New())

	decrypt := true
	paramsIn := ssm.GetParametersInput{
		Names: []*string{
			aws.String("buddybot-botToken"),
			aws.String("buddybot-usrToken"),
			aws.String("buddybot-reqSecret"),
			aws.String("buddybot-clientID"),
			aws.String("buddybot-clientSecret"),
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

	b.ClientID = params["buddybot-clientID"]
	if b.ClientID == "" {
		return nil, errors.New("required parameter 'buddybot-clientID' is undefined")
	}

	b.ClientSecret = params["buddybot-clientSecret"]
	if b.ClientSecret == "" {
		return nil, errors.New("required parameter 'buddybot-clientSecret' is undefined")
	}

	b.BotToken = params["buddybot-botToken"]
	if b.BotToken == "" {
		return nil, errors.New("required parameter 'buddybot-botToken' is undefined")
	}

	b.UsrToken = params["buddybot-usrToken"]
	if b.UsrToken == "" {
		return nil, errors.New("required parameter 'buddybot-usrToken' is undefined")
	}

	b.ReqSecret = params["buddybot-reqSecret"]
	if b.ReqSecret == "" {
		return nil, errors.New("required parameter 'buddybot-reqSecret' is undefined")
	}

	b.Region = os.Getenv("BUDDYBOT_REGION")
	if b.Region == "" {
		return nil, errors.New("required environment variable  'BUDDYBOT_REGION' is undefined")
	}

	b.AuthTable = os.Getenv("BUDDYBOT_AUTH_TABLE")
	if b.AuthTable == "" {
		return nil, errors.New("required environment variable  'BUDDYBOT_AUTH_TABLE' is undefined")
	}

	b.ScoreTable = os.Getenv("BUDDYBOT_SCORE_TABLE")
	if b.ScoreTable == "" {
		return nil, errors.New("required environment variable  'BUDDYBOT_SCORE_TABLE' is undefined")
	}

	return b, nil
}

// ParseSlashCommand takes an AWS API Gateway Request and returns a Slack SlashCommand. It returns an error
// if the request is invalid or it is unable to parse the request.
func (b *SlackBot) ParseSlashCommand(req events.APIGatewayProxyRequest) (slack.SlashCommand, error) {
	s := slack.SlashCommand{}

	err := b.validateRequest(req)
	if err != nil {
		return s, err
	}

	v, err := url.ParseQuery(req.Body)
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

// ParseEvent takes an AWS API Gateway Request and returns a Slack EventsAPIEvent. It returns an error
// if the request is invalid or it is unable to parse the request.
func (b *SlackBot) ParseEvent(req events.APIGatewayProxyRequest) (slackevents.EventsAPIEvent, error) {
	e := slackevents.EventsAPIEvent{}

	err := b.validateRequest(req)
	if err != nil {
		return e, err
	}

	e, err = slackevents.ParseEvent(json.RawMessage(req.Body), slackevents.OptionVerifyToken(&NullComparator{}))
	if err != nil {
		return e, err
	}

	return e, nil
}

// ValidateRequest returns an error if the request doesn't pass validation. Validation is
// performed against the method, headers and signature. It returns nil if the request is valid.
func (b *SlackBot) validateRequest(req events.APIGatewayProxyRequest) error {
	// we expect all requests to be POST requests
	if req.HTTPMethod != http.MethodPost {
		return errors.New(fmt.Sprintf("invalid method %s", req.HTTPMethod))
	}

	// we expect Slack headers to be populated
	ts := req.Headers["X-Slack-Request-Timestamp"]
	if len(ts) < 10 {
		return errors.New("invalid/no 'X-Slack-Request-Timestamp' header specified")
	}

	sig := req.Headers["X-Slack-Signature"]
	if len(sig) < 3 {
		return errors.New("invalid/no 'X-Slack-Signature' header specified")
	}

	// validate the request signature is correct before handling the event
	valid := checkHMAC(req.Body, ts, sig[3:], b.ReqSecret)
	if valid != true {
		return errors.New("failed to validate request signature")
	}

	return nil
}

// RetrieveToken queries the AuthTable to identify the auth token for a given
// Slack team. It returns an error if it is unable to find the token.
func (b *SlackBot) RetrieveToken(teamID string) (string, error) {
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

// CheckHMAC reports whether msgHMAC is a valid HMAC tag for msg.
func checkHMAC(body, timestamp, msgHMAC, key string) bool {
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

// AuthRecord represents the access token we store in DynamoDB for
// every authenticated workspace.
// TODO: consider whether this should go in a separate records package
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
