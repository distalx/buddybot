package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"

	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/billglover/buddybot/bot"
)

// AuthResponse represents the response we receive from Slack when
// a user adds our app to their workspace.
type AuthResponse struct {
	Ok          bool   `json:"ok"`
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	UserID      string `json:"user_id"`
	TeamName    string `json:"team_name"`
	TeamID      string `json:"team_id"`
	Bot         struct {
		BotUserID      string `json:"bot_user_id"`
		BotAccessToken string `json:"bot_access_token"`
	} `json:"bot"`
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

func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	fmt.Println("INFO:", req.HTTPMethod, req.Path)
	fmt.Println("INFO:", req.QueryStringParameters)

	b, err := bot.New()
	if err != nil {
		fmt.Println("ERROR: unable to initiate the bot:", err)
		os.Exit(1)
	}

	// change the temporary code for an API access token
	v := url.Values{}
	v.Set("code", req.QueryStringParameters["code"])
	v.Set("name", "https://k1jenua1ml.execute-api.eu-west-1.amazonaws.com/Prod/auth")

	r, err := http.NewRequest(http.MethodPost, "https://slack.com/api/oauth.access", strings.NewReader(v.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth(b.ClientID, b.ClientSecret)
	client := http.DefaultClient
	resp, err := client.Do(r)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		fmt.Println("ERROR: unable to request auth token:", err)
		apiResp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
		return apiResp, nil
	}

	ar := new(AuthResponse)
	err = json.NewDecoder(resp.Body).Decode(ar)
	if err != nil {
		fmt.Println("ERROR: unable to decode auth token:", err)
		apiResp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
		return apiResp, nil
	}

	fmt.Println("INFO:", ar)

	// store web-hook payload in DynamoDB
	record := AuthRecord{
		UID:            ar.TeamID,
		UserID:         ar.UserID,
		AccessToken:    ar.AccessToken,
		Scope:          ar.Scope,
		TeamName:       ar.TeamName,
		TeamID:         ar.TeamID,
		BotUserID:      ar.Bot.BotUserID,
		BotAccessToken: ar.Bot.BotAccessToken,
	}
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(b.Region)},
	)

	if err != nil {
		fmt.Println("ERROR: unable to create DynamoDB session:", err)
		apiResp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
		return apiResp, nil
	}

	ddb := dynamodb.New(sess)

	payload, err := dynamodbattribute.MarshalMap(record)
	if err != nil {
		fmt.Println("ERROR: unable to marshal DynamoDB record:", err)
		apiResp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
		return apiResp, nil
	}

	input := &dynamodb.PutItemInput{
		Item:      payload,
		TableName: aws.String(b.AuthTable),
	}

	_, err = ddb.PutItem(input)
	if err != nil {
		fmt.Println("ERROR: unable to put record in DynamoDB:", err)
		apiResp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
		return apiResp, nil
	}

	fmt.Println("INFO: successfully put record in DynamoDB:")

	pageBuf := new(bytes.Buffer)
	t := template.Must(template.New("t1").
		Parse("<html><body><h1>BuddyBot</h1><p>Successfully authenticated for: {{.}}</p></body></html>"))
	err = t.Execute(pageBuf, record.TeamName)
	if err != nil {
		fmt.Println("ERROR: unable to render template:", err)
		apiResp := events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}
		return apiResp, nil
	}

	apiResp := events.APIGatewayProxyResponse{
		Body:       pageBuf.String(),
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
	}

	return apiResp, nil
}

func main() {
	lambda.Start(handler)
}
