package plusplus

import (
	"log"
	"regexp"

	"github.com/nlopes/slack/slackevents"
)

// Bot represents a single instance of BuddyBot
type Bot struct{}

// Start starts an instance  the bot and returns an events channel where it will read
func (b *Bot) Start(evtChan <-chan slackevents.EventsAPIEvent) {
	for e := range evtChan {
		log.Println("handling:", e.Type)
	}
}

// IdentifyPlusPlus takes a message and returns a slice of users tagged for PlusPlus.
func identifyPlusPlus(msg string) []string {
	var users []string
	var re = regexp.MustCompile(`(?m)\<\@(\w+)\>\+\+`)
	for _, match := range re.FindAllStringSubmatch(msg, -1) {
		users = append(users, string(match[1]))
	}
	return users
}

// IdentifyMinusMinus takes a message and returns a slice of users tagged for MinusMinus.
func identifyMinusMinus(msg string) []string {
	var users []string
	var re = regexp.MustCompile(`(?m)\<\@(\w+)\>\-\-`)
	for _, match := range re.FindAllStringSubmatch(msg, -1) {
		users = append(users, string(match[1]))
	}
	return users
}
