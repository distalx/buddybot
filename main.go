package main

import (
	"fmt"

	"github.com/nlopes/slack"
)

func main() {
	api := slack.New("TOKEN_HERE")
	//api.SetDebug(true)

	users, err := api.GetUsers()
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	for _, user := range users {
		fmt.Printf("ID: %s, Name: %s\n", user.ID, user.Name)
	}
}
