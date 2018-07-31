package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	b, err := newBot()
	if err != nil {
		fmt.Println("ERROR: unable to initiate the bot:", err)
		os.Exit(1)
	}

	lambda.Start(b.handler)
}
