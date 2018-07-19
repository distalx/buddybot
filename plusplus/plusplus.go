package plusplus

import (
	"log"
	"regexp"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

// Bot represents a single instance of the Bot
type Bot struct {
	token string // auth token
	uid   string // user id
	tid   string // team id
}

// New returns a new instance of PlusPlus
func New(token string) (*Bot, error) {
	b := new(Bot)
	b.token = token

	api := slack.New(token)
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
		log.Println("handling:", e.Type)
		log.Printf("%+v", e)

		switch ie := e.InnerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			log.Println("message text:", ie.Text)
			log.Println("message subtype:", ie.SubType)
			log.Printf("%+v", ie)
		default:
			log.Println("unhandled callback type:", e.InnerEvent.Type)
		}
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
