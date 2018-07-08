package main

import (
	"fmt"
	"os"

	"github.com/nlopes/slack"
)

func main() {
	println("BuddyBot")

	token := os.Getenv("BUDDYBOT_TOKEN")
	if token == "" {
		fmt.Println("Token must be provided by setting the BUDDYBOT_TOKEN EnvVar")
		os.Exit(1)
	}

	api := slack.New(token)
	api.SetDebug(false)

	// list all users in the Slack workspace
	users, err := api.GetUsers()
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	for _, user := range users {
		fmt.Printf("ID: %s, Name: %s\n", user.ID, user.Name)
	}
}
