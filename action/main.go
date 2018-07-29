package main

import (
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	fmt.Println("INFO:", req.HTTPMethod, req.Path)

	resp := events.APIGatewayProxyResponse{StatusCode: http.StatusAccepted}
	return resp, nil
}

func main() {
	lambda.Start(handler)
}
