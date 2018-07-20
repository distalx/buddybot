package plusplus

import (
	"fmt"
	"log"
	"regexp"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

// Bot represents a single instance of the Bot
type Bot struct {
	auth_token string // auth token - typically for writing messages
	user_token string // user token - typically for reading messages
	uid        string // user id
	tid        string // team id
}

// New returns a new instance of PlusPlus
func New(auth_token, user_token string) (*Bot, error) {
	b := new(Bot)
	b.auth_token = auth_token
	b.user_token = user_token

	api := slack.New(b.auth_token)
	atr, err := api.AuthTest()
	if err != nil {
		return nil, err
	}

	log.Println("Name:", atr.Team)

	b.uid = atr.UserID
	log.Println("UID:", b.uid)

	b.tid = atr.TeamID
	log.Println("TID:", b.tid)

	return b, nil
}

// Start starts an instance  the bot and returns an events channel where it will read
func (b *Bot) Start(evtChan <-chan slackevents.EventsAPIEvent) {
	for e := range evtChan {

		switch ie := e.InnerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			if ie.SubType != "" {
				log.Println("skipping scoring because:", ie.SubType)
				break
			}
			err := b.scoreMessage(ie)
			if err != nil {
				log.Println("unable to reply to message:", err)
			}

		default:
			log.Println("unhandled callback type:", e.InnerEvent.Type)
		}
	}
}

// scoreMessage takes a message and handles s
func (b *Bot) scoreMessage(msg *slackevents.MessageEvent) error {
	plusUsers := identifyPlusPlus(msg.Text)
	for _, u := range plusUsers {
		api := slack.New(b.auth_token)
		params := slack.PostMessageParameters{
			Username:        "UBLPTK0JH",
			AsUser:          true,
			ThreadTimestamp: msg.TimeStamp,
		}
		reply := fmt.Sprintf("congrats <@%s>! :smile:", u)
		_, _, err := api.PostMessage(msg.Channel, reply, params)
		if err != nil {
			return err
		}
	}

	minusUsers := identifyMinusMinus(msg.Text)
	for _, u := range minusUsers {
		api := slack.New(b.auth_token)
		params := slack.PostMessageParameters{
			Username:        "UBLPTK0JH",
			AsUser:          true,
			ThreadTimestamp: msg.TimeStamp,
		}
		reply := fmt.Sprintf("Commiserations <@%s>! :sob:", u)
		_, _, err := api.PostMessage(msg.Channel, reply, params)
		if err != nil {
			return err
		}
	}

	return nil
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
