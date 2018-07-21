package plusplus

import (
	"fmt"
	"log"
	"regexp"

	"github.com/billglover/buddybot/datastore"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

// Bot represents a single instance of the Bot
type Bot struct {
	authToken string // authToken - typically for writing messages
	userToken string // userToken - typically for reading messages
	ds        datastore.Scorer
	uid       string // user id
	tid       string // team id
}

// New returns a new instance of PlusPlus
func New(authToken, userToken string) (*Bot, error) {
	b := new(Bot)
	b.authToken = authToken
	b.userToken = userToken
	ds, err := datastore.New("score.db")
	if err != nil {
		return nil, err
	}
	b.ds = ds

	api := slack.New(b.authToken)
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
		api := slack.New(b.authToken)
		params := slack.PostMessageParameters{
			Username:        b.uid,
			AsUser:          true,
			ThreadTimestamp: msg.TimeStamp,
		}
		score, err := b.ds.Inc(b.tid, u)
		if err != nil {
			return err
		}

		reply := fmt.Sprintf("Congrats <@%s>! Score now at %d :smile:", u, score)
		_, _, err = api.PostMessage(msg.Channel, reply, params)
		if err != nil {
			return err
		}
	}

	minusUsers := identifyMinusMinus(msg.Text)
	for _, u := range minusUsers {
		api := slack.New(b.authToken)
		params := slack.PostMessageParameters{
			Username:        b.uid,
			AsUser:          true,
			ThreadTimestamp: msg.TimeStamp,
		}
		score, err := b.ds.Dec(b.tid, u)
		if err != nil {
			return err
		}

		reply := fmt.Sprintf("Commiserations <@%s>! Score now at %d :smile:", u, score)
		_, _, err = api.PostMessage(msg.Channel, reply, params)
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop closes the database and stops the bot
func (b *Bot) Stop() {
	err := b.ds.Close()
	if err != nil {
		log.Println("unable to stop bot:", err)
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
